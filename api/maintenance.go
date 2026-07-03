package main

import "net/http"

// withMaintenance 503s every route except health checks when on.
func withMaintenance(on bool, next http.Handler) http.Handler {
	if !on {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/livez" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Retry-After", "3600")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "maintenance"})
	})
}
