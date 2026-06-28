package main

import (
	"context"
	"testing"
)

func TestFakeGoogleVerifier_ParsesSentinel(t *testing.T) {
	v := fakeGoogleVerifier()
	c, err := v(context.Background(), "e2e:alice@gmail.com")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if c.Email != "alice@gmail.com" || !c.EmailVerified || c.Sub == "" {
		t.Fatalf("unexpected claims: %+v", c)
	}
}

func TestFakeGoogleVerifier_RejectsNonSentinel(t *testing.T) {
	v := fakeGoogleVerifier()
	if _, err := v(context.Background(), "not-a-sentinel"); err == nil {
		t.Fatal("expected error for a non-sentinel token, got nil")
	}
}

func TestSelectGoogleVerifier_PicksFakeWhenEnabled(t *testing.T) {
	cfg := Config{GoogleAuthFake: true}
	if _, err := selectGoogleVerifier(cfg)(context.Background(), "e2e:bob@gmail.com"); err != nil {
		t.Fatalf("fake verifier should accept sentinel: %v", err)
	}
}
