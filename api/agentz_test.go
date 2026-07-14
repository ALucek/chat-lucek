package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCandURLFor(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://chat-agent-abc-uc.a.run.app", "https://cand---chat-agent-abc-uc.a.run.app"},
		{"http://localhost:8081", "http://localhost:8081"},
	}
	for _, tc := range cases {
		if got := candURLFor(tc.in); got != tc.want {
			t.Fatalf("candURLFor(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAgentRun_DevRoutesToCand(t *testing.T) {
	var gotBase, gotCand bool
	var body map[string]any
	base := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		gotBase = true
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, endFrame)
	}))
	defer base.Close()
	cand := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCand = true
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, endFrame)
	}))
	defer cand.Close()
	c := &agentClient{baseURL: base.URL, candURL: cand.URL, http: cand.Client()}

	if _, err := c.run(context.Background(), nil, "", true, runHandlers{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotBase || !gotCand {
		t.Fatalf("dev run should hit cand only: base=%v cand=%v", gotBase, gotCand)
	}
	if body["dev"] != true {
		t.Fatalf("dev flag not set in payload: %v", body)
	}
}

func TestAgentz_Gating(t *testing.T) {
	// Off the dev host, and with no dev host configured, /agentz is 404.
	for _, dev := range []string{"dev.example.com", ""} {
		c := &Chat{devHost: dev}
		rec := httptest.NewRecorder()
		c.Agentz(rec, httptest.NewRequest("GET", "http://example.com/agentz", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("devHost=%q: want 404, got %d", dev, rec.Code)
		}
	}
}

func TestAgentz_OnDevHost(t *testing.T) {
	good := deltaFrame("a:text", "Hi") + usageFrame(1, 2) + endFrame
	cases := []struct {
		name   string
		frames string
		want   int
	}{
		{"ok", good, http.StatusOK},
		{"no usage", deltaFrame("a:text", "Hi") + endFrame, http.StatusServiceUnavailable},
		{"error", "event: error\ndata: {\"message\":\"boom\"}\n\n", http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, tc.frames)
			}))
			defer srv.Close()
			c := &Chat{devHost: "dev.example.com", agent: &agentClient{baseURL: srv.URL, candURL: srv.URL, http: srv.Client()}}
			rec := httptest.NewRecorder()
			c.Agentz(rec, httptest.NewRequest("GET", "http://dev.example.com/agentz", nil))
			if rec.Code != tc.want {
				t.Fatalf("%s: want %d, got %d", tc.name, tc.want, rec.Code)
			}
		})
	}
}
