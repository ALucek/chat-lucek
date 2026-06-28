package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"google.golang.org/api/idtoken"
)

// googleClaims is the subset of a verified Google ID token we use.
type googleClaims struct {
	Sub           string
	Email         string
	EmailVerified bool
}

// googleVerifier verifies a Google ID token and returns its claims.
type googleVerifier func(ctx context.Context, idToken string) (googleClaims, error)

// realGoogleVerifier validates the token against Google's keys with clientID as the audience.
func realGoogleVerifier(clientID string) googleVerifier {
	return func(ctx context.Context, idToken string) (googleClaims, error) {
		p, err := idtoken.Validate(ctx, idToken, clientID)
		if err != nil {
			return googleClaims{}, err
		}
		c := googleClaims{Sub: p.Subject}
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
		return googleClaims{Sub: "e2e:" + email, Email: email, EmailVerified: true}, nil
	}
}

// selectGoogleVerifier returns the fake verifier when GOOGLE_AUTH_FAKE is set, else the real one.
func selectGoogleVerifier(cfg Config) googleVerifier {
	if cfg.GoogleAuthFake {
		slog.Warn("GOOGLE_AUTH_FAKE enabled: accepting fake e2e tokens — never use in production")
		return fakeGoogleVerifier()
	}
	return realGoogleVerifier(cfg.GoogleClientID)
}
