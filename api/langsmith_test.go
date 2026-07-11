package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func fptr(f float64) *float64 { return &f }
func sptr(s string) *string   { return &s }

func TestLangsmithPost_Score(t *testing.T) {
	var gotKey, gotBody, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newLangsmithClient(srv.URL, "key-123")
	if err := c.post(context.Background(), "fb-1", "run-1", "user_score", fptr(1.0), nil); err != nil {
		t.Fatalf("post: %v", err)
	}
	if gotPath != "/api/v1/feedback" {
		t.Fatalf("path: %s", gotPath)
	}
	if gotKey != "key-123" {
		t.Fatalf("api key: %q", gotKey)
	}
	for _, want := range []string{`"id":"fb-1"`, `"run_id":"run-1"`, `"key":"user_score"`, `"score":1`} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("body missing %s: %s", want, gotBody)
		}
	}
	if strings.Contains(gotBody, "value") {
		t.Fatalf("score feedback should carry no value: %s", gotBody)
	}
}

func TestLangsmithPost_Value(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := newLangsmithClient(srv.URL, "k")
	if err := c.post(context.Background(), "fb-2", "run-1", "user_feedback", nil, sptr("clear")); err != nil {
		t.Fatalf("post: %v", err)
	}
	for _, want := range []string{`"key":"user_feedback"`, `"value":"clear"`} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("body missing %s: %s", want, gotBody)
		}
	}
	if strings.Contains(gotBody, "score") {
		t.Fatalf("comment feedback should carry no score: %s", gotBody)
	}
}

func TestLangsmithPost_Non2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if err := newLangsmithClient(srv.URL, "k").post(context.Background(), "f", "r", "user_score", fptr(0.0), nil); err == nil {
		t.Fatal("want error on 500")
	}
}

func TestLangsmithDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	if err := newLangsmithClient(srv.URL, "k").del(context.Background(), "fb-9"); err != nil {
		t.Fatalf("del: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/v1/feedback/fb-9" {
		t.Fatalf("del request: %s %s", gotMethod, gotPath)
	}
}

func TestLangsmithDelete_404IsOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if err := newLangsmithClient(srv.URL, "k").del(context.Background(), "gone"); err != nil {
		t.Fatalf("404 should be treated as absent, got %v", err)
	}
}

func TestLangsmithEnabled(t *testing.T) {
	if newLangsmithClient("e", "").enabled() {
		t.Fatal("empty key should be disabled")
	}
	if !newLangsmithClient("e", "k").enabled() {
		t.Fatal("key should be enabled")
	}
}

func TestNewUUID_Format(t *testing.T) {
	u, err := newUUID()
	if err != nil {
		t.Fatalf("newUUID: %v", err)
	}
	re := `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`
	if ok, _ := regexp.MatchString(re, u); !ok {
		t.Fatalf("bad uuid: %s", u)
	}
}

func TestDeriveUUID_Deterministic(t *testing.T) {
	a := deriveUUID("base-id", "user_feedback")
	if a != deriveUUID("base-id", "user_feedback") {
		t.Fatal("deriveUUID should be stable for the same inputs")
	}
	if a == deriveUUID("base-id", "other") {
		t.Fatal("deriveUUID should differ by name")
	}
	re := `^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`
	if ok, _ := regexp.MatchString(re, a); !ok {
		t.Fatalf("bad v5 uuid: %s", a)
	}
}
