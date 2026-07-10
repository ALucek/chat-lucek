package main

import "strings"

// canonicalizeEmail returns the abuse-accounting key, not the account identity.
func canonicalizeEmail(s string) string {
	e := normalizeEmail(s)
	at := strings.LastIndex(e, "@")
	if at < 0 {
		return e
	}
	local, domain := e[:at], e[at+1:]
	if i := strings.IndexByte(local, '+'); i >= 0 {
		local = local[:i]
	}
	if domain == "googlemail.com" {
		domain = "gmail.com"
	}
	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}
	return local + "@" + domain
}
