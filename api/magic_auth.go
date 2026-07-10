package main

import (
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

const (
	magicTokenTTL   = 15 * time.Minute
	magicRatePerMin = 5
	magicRateBurst  = 5
)

// clientIP returns the first X-Forwarded-For hop, else the connection host.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// MagicRequest emails a single-use sign-in link. Always 200 unless throttled.
func (a *Auth) MagicRequest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, err := mail.ParseAddress(body.Email); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "a valid email is required"})
		return
	}
	id := normalizeEmail(body.Email)
	key := canonicalizeEmail(body.Email)
	if ok, _ := a.magicLimiter.allow("ip:" + clientIP(r)); !ok {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		return
	}
	if ok, _ := a.magicLimiter.allow("email:" + key); !ok {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		return
	}

	sent := func() { writeJSON(w, http.StatusOK, map[string]string{"status": "sent"}) }
	if isDisposableEmail(id) {
		sent()
		return
	}
	var exists bool
	_ = a.pool.QueryRow(r.Context(),
		`select exists(select 1 from users where email = $1)`, id).Scan(&exists)
	if !exists && !a.signupOpen {
		sent()
		return
	}

	raw, err := newRefreshToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create link"})
		return
	}
	if _, err := a.pool.Exec(r.Context(),
		`insert into magic_links (token_hash, email, expires_at) values ($1, $2, $3)`,
		hashToken(raw), id, time.Now().Add(magicTokenTTL)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create link"})
		return
	}
	link := a.linkBase + "/auth/magic?token=" + url.QueryEscape(raw)
	if err := a.mailer.SendMagicLink(r.Context(), id, link); err != nil {
		slog.ErrorContext(r.Context(), "magic link send", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not send link"})
		return
	}
	sent()
}

// MagicVerify claims a single-use token and starts a session for its email.
func (a *Auth) MagicVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token required"})
		return
	}
	var email string
	err := a.pool.QueryRow(r.Context(),
		`delete from magic_links where token_hash = $1 and expires_at > now() returning email`,
		hashToken(body.Token)).Scan(&email)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired link"})
		return
	}
	a.completeLogin(w, r, email)
}

// MagicLatest returns the last captured link; only works with the fake mailer.
func (a *Auth) MagicLatest(w http.ResponseWriter, r *http.Request) {
	fm, ok := a.mailer.(*fakeMailer)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	link, ok := fm.last(normalizeEmail(r.URL.Query().Get("email")))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no link"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"link": link})
}
