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

func TestLangsmithPost_SendsFeedback(t *testing.T) {
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
	if err := c.post(context.Background(), "fb-1", "run-1", 1.0, "nice"); err != nil {
		t.Fatalf("post: %v", err)
	}
	if gotPath != "/feedback" {
		t.Fatalf("path: %s", gotPath)
	}
	if gotKey != "key-123" {
		t.Fatalf("api key: %q", gotKey)
	}
	for _, want := range []string{`"id":"fb-1"`, `"run_id":"run-1"`, `"key":"user_score"`, `"score":1`, `"comment":"nice"`} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("body missing %s: %s", want, gotBody)
		}
	}
}

func TestLangsmithPost_OmitsEmptyComment(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := newLangsmithClient(srv.URL, "k")
	if err := c.post(context.Background(), "f", "r", 0.0, ""); err != nil {
		t.Fatalf("post: %v", err)
	}
	if strings.Contains(gotBody, "comment") {
		t.Fatalf("comment should be omitted: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"score":0`) {
		t.Fatalf("want score 0: %s", gotBody)
	}
}

func TestLangsmithPost_Non2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if err := newLangsmithClient(srv.URL, "k").post(context.Background(), "f", "r", 0.0, ""); err == nil {
		t.Fatal("want error on 500")
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
