package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/api/idtoken"
)

// googleTokenURL is Google's OAuth2 token endpoint; overridable in tests.
var googleTokenURL = "https://oauth2.googleapis.com/token"

// googleClaims is the subset of a verified Google ID token we use.
type googleClaims struct {
	Email         string
	EmailVerified bool
}

// googleVerifier verifies a Google ID token and returns its claims.
type googleVerifier func(ctx context.Context, idToken string) (googleClaims, error)

// realGoogleVerifier validates the token against Google's keys (aud=clientID).
func realGoogleVerifier(clientID string) googleVerifier {
	return func(ctx context.Context, idToken string) (googleClaims, error) {
		p, err := idtoken.Validate(ctx, idToken, clientID)
		if err != nil {
			return googleClaims{}, err
		}
		c := googleClaims{}
		if e, ok := p.Claims["email"].(string); ok {
			c.Email = e
		}
		if v, ok := p.Claims["email_verified"].(bool); ok {
			c.EmailVerified = v
		}
		return c, nil
	}
}

// fakeGoogleVerifier accepts sentinel "e2e:<email>" tokens. Test-only.
func fakeGoogleVerifier() googleVerifier {
	return func(_ context.Context, idToken string) (googleClaims, error) {
		email, ok := strings.CutPrefix(idToken, "e2e:")
		if !ok || email == "" {
			return googleClaims{}, errors.New("fake verifier: expected e2e:<email>")
		}
		return googleClaims{Email: email, EmailVerified: true}, nil
	}
}

// selectGoogleVerifier: fake verifier when GOOGLE_AUTH_FAKE, else real.
func selectGoogleVerifier(cfg Config) googleVerifier {
	if cfg.GoogleAuthFake {
		slog.Warn("GOOGLE_AUTH_FAKE enabled: accepting fake e2e tokens — never use in production")
		return fakeGoogleVerifier()
	}
	return realGoogleVerifier(cfg.GoogleClientID)
}

// googleExchanger trades a one-time auth code for a Google ID token.
type googleExchanger func(ctx context.Context, code string) (idToken string, err error)

// realGoogleExchanger exchanges the popup auth code at Google's token endpoint.
func realGoogleExchanger(clientID, clientSecret string) googleExchanger {
	return func(ctx context.Context, code string) (string, error) {
		form := url.Values{
			"code":          {code},
			"client_id":     {clientID},
			"client_secret": {clientSecret},
			"redirect_uri":  {"postmessage"},
			"grant_type":    {"authorization_code"},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			googleTokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("token exchange failed: %s", resp.Status)
		}
		var out struct {
			IDToken string `json:"id_token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return "", err
		}
		if out.IDToken == "" {
			return "", errors.New("token exchange: no id_token in response")
		}
		return out.IDToken, nil
	}
}

// fakeGoogleExchanger passes the code through as the id token. Test-only.
func fakeGoogleExchanger() googleExchanger {
	return func(_ context.Context, code string) (string, error) { return code, nil }
}

// selectGoogleExchanger: fake exchanger when GOOGLE_AUTH_FAKE, else real.
func selectGoogleExchanger(cfg Config) googleExchanger {
	if cfg.GoogleAuthFake {
		return fakeGoogleExchanger()
	}
	return realGoogleExchanger(cfg.GoogleClientID, cfg.GoogleClientSecret)
}

// Google exchanges the code, verifies it, then starts a session by email.
func (a *Auth) Google(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code required"})
		return
	}
	idToken, err := a.exchange(r.Context(), body.Code)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid google token"})
		return
	}
	claims, err := a.verify(r.Context(), idToken)
	if err != nil || !claims.EmailVerified || claims.Email == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid google token"})
		return
	}
	a.completeLogin(w, r, claims.Email)
}
