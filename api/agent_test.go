package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentRun_DispatchesEventsAndUsage(t *testing.T) {
	frames := "event: reasoning\ndata: {\"text\":\"thinking\"}\n\n" +
		"event: status\ndata: {\"id\":\"r1\",\"kind\":\"search\",\"detail\":\"cats\",\"state\":\"start\"}\n\n" +
		tokenFrame("Hello") + tokenFrame(" there") +
		usageFrame(4, 6) + endFrame
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, frames)
	}))
	defer srv.Close()
	c := &agentClient{baseURL: srv.URL, http: srv.Client()}

	var text, reasoning string
	var statuses []statusEvent
	usage, err := c.run(context.Background(), []llmMessage{{Role: "user", Content: "hi"}}, runHandlers{
		onToken:     func(s string) { text += s },
		onReasoning: func(s string) { reasoning += s },
		onStatus:    func(s statusEvent) { statuses = append(statuses, s) },
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if text != "Hello there" {
		t.Fatalf("text: %q", text)
	}
	if reasoning != "thinking" {
		t.Fatalf("reasoning: %q", reasoning)
	}
	if len(statuses) != 1 || statuses[0].Kind != "search" || statuses[0].Detail != "cats" {
		t.Fatalf("statuses: %+v", statuses)
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

	if _, err := c.run(context.Background(), nil, runHandlers{}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want error containing boom, got %v", err)
	}
}

func TestAgentRun_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &agentClient{baseURL: srv.URL, http: srv.Client()}

	if _, err := c.run(context.Background(), nil, runHandlers{}); err == nil {
		t.Fatal("want error on non-200 status")
	}
}
