package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithCORS_SetsHeadersAndCallsThrough(t *testing.T) {
	called := false
	h := withCORS([]string{"http://localhost:3000"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/me", nil))

	if !called {
		t.Fatal("wrapped handler should be called for non-preflight requests")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("allow-origin: want http://localhost:3000, got %q", got)
	}
}

func TestWithCORS_EchoesAllowedOriginWithVary(t *testing.T) {
	allowed := []string{"https://chat.lucek.ai", "https://dev.chat.lucek.ai"}
	h := withCORS(allowed, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	r.Header.Set("Origin", "https://dev.chat.lucek.ai")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://dev.chat.lucek.ai" {
		t.Fatalf("allow-origin: want the dev origin echoed, got %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("vary: want Origin, got %q", got)
	}
}

func TestWithCORS_FallsBackToPrimaryForUnlistedOrigin(t *testing.T) {
	allowed := []string{"https://chat.lucek.ai", "https://dev.chat.lucek.ai"}
	h := withCORS(allowed, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	r.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://chat.lucek.ai" {
		t.Fatalf("allow-origin: want primary fallback, got %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "" {
		t.Fatalf("vary: want unset for unlisted origin, got %q", got)
	}
}

func TestWithCORS_PreflightShortCircuits(t *testing.T) {
	called := false
	h := withCORS([]string{"http://localhost:3000"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/api/google", nil))

	if called {
		t.Fatal("preflight OPTIONS must not reach the wrapped handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight: want 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("preflight must advertise allowed request headers")
	}
}
