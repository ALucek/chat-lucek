package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentRun_DispatchesNodesAndUsage(t *testing.T) {
	frames := nodeStartFrame("r1:reasoning", "", "reasoning", "", "") +
		deltaFrame("r1:reasoning", "thinking") +
		nodeStartFrame("s1", "SA", "tool", "internet_search", `{"query":"cats"}`) +
		nodeEndFrame("s1", `{"results":[]}`) +
		nodeStartFrame("a:text", "", "text", "", "") +
		deltaFrame("a:text", "Hello") + deltaFrame("a:text", " there") +
		usageFrame(4, 6) + endFrame
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, frames)
	}))
	defer srv.Close()
	c := &agentClient{baseURL: srv.URL, http: srv.Client()}

	var nodes []nodeFrame
	deltas := map[string]string{}
	ends := map[string]string{}
	usage, err := c.run(context.Background(), []llmMessage{{Role: "user", Content: "hi"}}, "", false, runHandlers{
		onNode:    func(n nodeFrame) { nodes = append(nodes, n) },
		onDelta:   func(id, text string) { deltas[id] += text },
		onNodeEnd: func(id string, out json.RawMessage) { ends[id] = string(out) },
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("want 3 nodes, got %d: %+v", len(nodes), nodes)
	}
	// the search nests under the subagent
	if nodes[1].ParentID == nil || *nodes[1].ParentID != "SA" || nodes[1].Name != "internet_search" {
		t.Fatalf("search node: %+v", nodes[1])
	}
	if deltas["a:text"] != "Hello there" {
		t.Fatalf("answer text: %q", deltas["a:text"])
	}
	if ends["s1"] != `{"results":[]}` {
		t.Fatalf("search output: %s", ends["s1"])
	}
	if usage.Prompt != 4 || usage.Completion != 6 {
		t.Fatalf("usage: %+v", usage)
	}
}

func TestAgentRun_ErrorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: error\ndata: {\"message\":\"boom\"}\n\n")
	}))
	defer srv.Close()
	c := &agentClient{baseURL: srv.URL, http: srv.Client()}

	if _, err := c.run(context.Background(), nil, "", false, runHandlers{}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want error containing boom, got %v", err)
	}
}

func TestAgentRun_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &agentClient{baseURL: srv.URL, http: srv.Client()}

	if _, err := c.run(context.Background(), nil, "", false, runHandlers{}); err == nil {
		t.Fatal("want error on non-200 status")
	}
}

func TestAgentRun_AttachesBearerOnlyWhenTokenSet(t *testing.T) {
	cases := []struct {
		name     string
		token    func(context.Context) (string, error)
		wantAuth string
	}{
		{"token set", func(context.Context) (string, error) { return "tok123", nil }, "Bearer tok123"},
		{"token nil", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, endFrame)
			}))
			defer srv.Close()
			c := &agentClient{baseURL: srv.URL, http: srv.Client(), token: tc.token}
			if _, err := c.run(context.Background(), nil, "", false, runHandlers{}); err != nil {
				t.Fatalf("run: %v", err)
			}
			if gotAuth != tc.wantAuth {
				t.Fatalf("auth header = %q, want %q", gotAuth, tc.wantAuth)
			}
		})
	}
}

func TestAgentRun_SendsThreadID(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, endFrame)
	}))
	defer srv.Close()
	c := &agentClient{baseURL: srv.URL, http: srv.Client()}

	if _, err := c.run(context.Background(), nil, "42", false, runHandlers{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if body["thread_id"] != "42" {
		t.Fatalf("thread_id: got %v, want 42", body["thread_id"])
	}

	body = nil
	if _, err := c.run(context.Background(), nil, "", false, runHandlers{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, ok := body["thread_id"]; ok {
		t.Fatalf("thread_id should be omitted when empty: %v", body)
	}
}
