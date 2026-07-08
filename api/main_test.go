package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadyHandler_OK(t *testing.T) {
	h := readyHandler(func(context.Context) error { return nil })
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestReadyHandler_DBDown(t *testing.T) {
	h := readyHandler(func(context.Context) error { return errors.New("db down") })
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := withSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	want := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "no-referrer",
		"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Permissions-Policy":        "camera=(), microphone=(), geolocation=()",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s: want %q, got %q", k, v, got)
		}
	}
}

func TestNewServer_Timeouts(t *testing.T) {
	srv := newServer(":8080", nil)
	// ReadHeaderTimeout guards against Slowloris; WriteTimeout must stay off so
	// long-lived SSE streams aren't cut. Exact durations are tuning, not contract.
	if srv.ReadHeaderTimeout <= 0 {
		t.Errorf("ReadHeaderTimeout: want >0 for Slowloris protection, got %v", srv.ReadHeaderTimeout)
	}
	if srv.WriteTimeout != 0 {
		t.Errorf("WriteTimeout: want 0 (off for SSE), got %v", srv.WriteTimeout)
	}
}
