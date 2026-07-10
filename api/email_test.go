package main

import "testing"

func TestCanonicalizeEmail(t *testing.T) {
	cases := map[string]string{
		" Adam+Work@Gmail.com ":  "adam@gmail.com",
		"a.d.am@googlemail.com":  "adam@gmail.com",
		"user+promo@outlook.com": "user@outlook.com",
		"first.last@company.com": "first.last@company.com",
		"PLAIN@Example.com":      "plain@example.com",
		"nodomain":               "nodomain",
	}
	for in, want := range cases {
		if got := canonicalizeEmail(in); got != want {
			t.Errorf("canonicalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
