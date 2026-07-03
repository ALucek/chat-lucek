package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithMaintenance(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		on         bool
		path       string
		wantStatus int
	}{
		{"off passes app route", false, "/api/me", http.StatusOK},
		{"on blocks app route", true, "/api/me", http.StatusServiceUnavailable},
		{"on allows livez", true, "/livez", http.StatusOK},
		{"on allows readyz", true, "/readyz", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := withMaintenance(tt.on, next)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("path %s on=%v: got %d want %d", tt.path, tt.on, rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusServiceUnavailable && rec.Header().Get("Retry-After") == "" {
				t.Errorf("503 response missing Retry-After header")
			}
		})
	}
}
