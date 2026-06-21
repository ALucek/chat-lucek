package main

import "testing"

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := hashPassword("password123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "password123" {
		t.Fatal("hash must not equal plaintext")
	}
	if err := checkPassword(hash, "password123"); err != nil {
		t.Fatalf("correct password should verify: %v", err)
	}
	if err := checkPassword(hash, "wrong"); err == nil {
		t.Fatal("wrong password should fail")
	}
}
