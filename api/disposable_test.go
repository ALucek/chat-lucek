package main

import "testing"

func TestIsDisposableEmail(t *testing.T) {
	if !isDisposableEmail("throwaway@mailinator.com") {
		t.Error("mailinator.com should be disposable")
	}
	if isDisposableEmail("real@gmail.com") {
		t.Error("gmail.com should not be disposable")
	}
	if isDisposableEmail("nodomain") {
		t.Error("malformed address should not match")
	}
}
