package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStream_SkipsCommentsAndStopsAtDone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, ": OPENROUTER PROCESSING\n\n") // keep-alive comment
		fmt.Fprint(w, deltaFrame("Hel"))
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{}}]}\n\n") // empty content
		fmt.Fprint(w, deltaFrame("lo"))
		fmt.Fprint(w, "data: [DONE]\n\n")
		fmt.Fprint(w, deltaFrame("AFTER")) // must not be delivered
	}))
	defer srv.Close()
	client := &openRouterClient{baseURL: srv.URL, http: srv.Client()}

	var got strings.Builder
	if err := client.stream(context.Background(), nil, func(s string) { got.WriteString(s) }); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if got.String() != "Hello" {
		t.Fatalf("want %q, got %q", "Hello", got.String())
	}
}

func TestStream_Non200IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := &openRouterClient{baseURL: srv.URL, http: srv.Client()}
	if err := client.stream(context.Background(), nil, func(string) {}); err == nil {
		t.Fatal("want error on non-200, got nil")
	}
}
