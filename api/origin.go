package main

import "net/http"

// originAllowed reports whether o is one of the configured browser origins.
func originAllowed(allowed []string, o string) bool {
	for _, a := range allowed {
		if o == a {
			return true
		}
	}
	return false
}

// withOriginCheck blocks unsafe requests with a mismatched Origin (CSRF).
func withOriginCheck(allowed []string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPatch, http.MethodDelete:
			if o := r.Header.Get("Origin"); o != "" && !originAllowed(allowed, o) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "origin not allowed"})
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}
