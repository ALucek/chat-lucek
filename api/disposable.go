package main

import (
	"strings"

	"github.com/lindell/go-burner-email-providers/burner"
)

// isDisposableEmail reports whether the address uses a burner/throwaway domain.
func isDisposableEmail(email string) bool {
	// The lib flags any input lacking a real domain; guard non-emails first.
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	return burner.IsBurnerEmail(email)
}
