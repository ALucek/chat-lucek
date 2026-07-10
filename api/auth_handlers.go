package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

const refreshTokenTTL = 30 * 24 * time.Hour

const refreshCookieName = "refresh_token"

// refreshCookie builds the cookie: Path=/api, SameSite=Strict (auth only).
func refreshCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     refreshCookieName,
		Value:    value,
		Path:     "/api",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}

func (a *Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookieName)
	if err != nil || c.Value == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
		return
	}
	h := hashToken(c.Value)

	// Claim: authorize and consume the token
	var userID int64
	var familyID string
	err = a.pool.QueryRow(r.Context(),
		`update refresh_tokens set revoked = true
		 where token_hash = $1 and not revoked and expires_at > now()
		 returning user_id, family_id`, h).Scan(&userID, &familyID)
	if err != nil {
		// Not claimable. treat as theft and revoke its whole family.
		var reusedFamily string
		if a.pool.QueryRow(r.Context(),
			`select family_id from refresh_tokens where token_hash = $1 and revoked`, h).
			Scan(&reusedFamily) == nil {
			_, _ = a.pool.Exec(r.Context(),
				`update refresh_tokens set revoked = true where family_id = $1`, reusedFamily)
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
		return
	}

	// Rotate: issue a fresh access + refresh token in the same family.
	a.purgeExpiredRefreshTokens(r.Context())
	a.issueTokens(w, r, userID, familyID, http.StatusOK)
}

// purgeExpiredRefreshTokens best-effort deletes expired tokens to cap growth.
func (a *Auth) purgeExpiredRefreshTokens(ctx context.Context) {
	_, _ = a.pool.Exec(ctx, `delete from refresh_tokens where expires_at < now()`)
}

func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookieName); err == nil && c.Value != "" {
		_, _ = a.pool.Exec(r.Context(),
			`delete from refresh_tokens where token_hash = $1`, hashToken(c.Value))
	}
	http.SetCookie(w, refreshCookie("", -1)) // clear it
	w.WriteHeader(http.StatusNoContent)      // idempotent: always 204
}

func (a *Auth) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		return
	}
	var email string
	if err := a.pool.QueryRow(r.Context(),
		`select email from users where id = $1`, userID).Scan(&email); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unknown user"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": userID, "email": email})
}

// completeLogin finds-or-creates the user by email and issues a session.
func (a *Auth) completeLogin(w http.ResponseWriter, r *http.Request, verifiedEmail string) {
	email := normalizeEmail(verifiedEmail)
	var userID int64
	err := a.pool.QueryRow(r.Context(),
		`select id from users where email = $1`, email).Scan(&userID)
	switch {
	case err == nil:
	case errors.Is(err, pgx.ErrNoRows):
		if !a.signupOpen {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "signups are closed"})
			return
		}
		if err := a.pool.QueryRow(r.Context(),
			`insert into users (email) values ($1) returning id`, email).Scan(&userID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create user"})
			return
		}
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create user"})
		return
	}
	family, err := newFamilyID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not start session"})
		return
	}
	a.issueTokens(w, r, userID, family, http.StatusOK)
}

// issueTokens mints an access token + a hashed refresh token, writes both.
func (a *Auth) issueTokens(w http.ResponseWriter, r *http.Request, userID int64, familyID string, status int) {
	access, err := mintAccessToken(a.secret, userID, time.Now())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not mint token"})
		return
	}
	raw, err := randomToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create refresh token"})
		return
	}
	_, err = a.pool.Exec(r.Context(),
		`insert into refresh_tokens (token_hash, user_id, family_id, expires_at) values ($1, $2, $3, $4)`,
		hashToken(raw), userID, familyID, time.Now().Add(refreshTokenTTL))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not store refresh token"})
		return
	}
	http.SetCookie(w, refreshCookie(raw, int(refreshTokenTTL.Seconds())))
	writeJSON(w, status, map[string]string{"access_token": access})
}
