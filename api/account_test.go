package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAccount_ExportScopedToCaller(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	ta, uidA := signup(t, mux, "a@x.com")
	tb, uidB := signup(t, mux, "b@x.com")
	ctx := context.Background()

	cid := createConversation(t, mux, ta)
	if _, err := testPool.Exec(ctx,
		`insert into messages (conversation_id, role, content, trace)
		 values ($1,'user','hi',null),($1,'assistant','yo','{"version":2}'::jsonb)`, cid); err != nil {
		t.Fatalf("seed messages: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`insert into token_usage (user_id, prompt_tokens, completion_tokens) values ($1,3,4)`, uidA); err != nil {
		t.Fatalf("seed usage: %v", err)
	}
	// B gets its own conversation to prove scoping.
	_ = createConversation(t, mux, tb)
	_ = uidB

	rec := do(t, mux, http.MethodGet, "/api/account/export", ta, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Fatalf("want attachment disposition, got %q", cd)
	}

	var out struct {
		Profile struct {
			Email string `json:"email"`
		} `json:"profile"`
		Conversations []struct {
			Messages []struct {
				Role  string          `json:"role"`
				Trace json.RawMessage `json:"trace"`
			} `json:"messages"`
		} `json:"conversations"`
		Usage []map[string]any `json:"usage"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Profile.Email != "a@x.com" {
		t.Fatalf("email: got %q", out.Profile.Email)
	}
	if len(out.Conversations) != 1 {
		t.Fatalf("want 1 conversation, got %d", len(out.Conversations))
	}
	if len(out.Conversations[0].Messages) != 2 {
		t.Fatalf("want 2 messages, got %d", len(out.Conversations[0].Messages))
	}
	if len(out.Usage) != 1 {
		t.Fatalf("want 1 usage row, got %d", len(out.Usage))
	}
}

func TestAccount_ExportRequiresAuth(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	rec := do(t, mux, http.MethodGet, "/api/account/export", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}
