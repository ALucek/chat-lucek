package main

import "net/http"

// withOriginCheck rejects state-changing requests whose Origin, when present,
// does not match allowedOrigin (CSRF defense-in-depth).
func withOriginCheck(allowedOrigin string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPatch, http.MethodDelete:
			if o := r.Header.Get("Origin"); o != "" && o != allowedOrigin {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "origin not allowed"})
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}
