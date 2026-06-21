package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestSignup_IssuesTokens(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	rec := do(t, mux, http.MethodPost, "/api/signup", "",
		map[string]string{"email": "a@x.com", "password": "password123"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", rec.Code)
	}
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatalf("missing tokens: %+v", out)
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	body := map[string]string{"email": "dup@x.com", "password": "password123"}
	do(t, mux, http.MethodPost, "/api/signup", "", body)
	rec := do(t, mux, http.MethodPost, "/api/signup", "", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", rec.Code)
	}
}

func TestLogin_OK(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	do(t, mux, http.MethodPost, "/api/signup", "",
		map[string]string{"email": "l@x.com", "password": "password123"})
	rec := do(t, mux, http.MethodPost, "/api/login", "",
		map[string]string{"email": "l@x.com", "password": "password123"})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	do(t, mux, http.MethodPost, "/api/signup", "",
		map[string]string{"email": "w@x.com", "password": "password123"})
	rec := do(t, mux, http.MethodPost, "/api/login", "",
		map[string]string{"email": "w@x.com", "password": "wrong"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	rec := do(t, mux, http.MethodPost, "/api/login", "",
		map[string]string{"email": "nobody@x.com", "password": "password123"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestMe_ReturnsUser(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	token, uid := signup(t, mux, "me@x.com")
	rec := do(t, mux, http.MethodGet, "/api/me", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var out struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.ID != uid || out.Email != "me@x.com" {
		t.Fatalf("unexpected user: %+v", out)
	}
}

func TestMe_NoToken(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	rec := do(t, mux, http.MethodGet, "/api/me", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestRefresh_Then_LogoutRevokes(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	rec := do(t, mux, http.MethodPost, "/api/signup", "",
		map[string]string{"email": "r@x.com", "password": "password123"})
	var tok struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tok); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if rr := do(t, mux, http.MethodPost, "/api/refresh", "",
		map[string]string{"refresh_token": tok.RefreshToken}); rr.Code != http.StatusOK {
		t.Fatalf("refresh: want 200, got %d", rr.Code)
	}
	if lo := do(t, mux, http.MethodPost, "/api/logout", "",
		map[string]string{"refresh_token": tok.RefreshToken}); lo.Code != http.StatusNoContent {
		t.Fatalf("logout: want 204, got %d", lo.Code)
	}
	if rr2 := do(t, mux, http.MethodPost, "/api/refresh", "",
		map[string]string{"refresh_token": tok.RefreshToken}); rr2.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: want 401, got %d", rr2.Code)
	}
}
