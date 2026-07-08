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
	if !strings.Contains(rec.Body.String(), "\n  ") {
		t.Fatalf("want indented (pretty) JSON")
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

func TestAccount_DeleteCascades(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	ta, uidA := signup(t, mux, "a@x.com")
	ctx := context.Background()

	cid := createConversation(t, mux, ta)
	if _, err := testPool.Exec(ctx,
		`insert into messages (conversation_id, role, content) values ($1,'user','hi')`, cid); err != nil {
		t.Fatalf("seed message: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`insert into token_usage (user_id, prompt_tokens, completion_tokens) values ($1,1,2)`, uidA); err != nil {
		t.Fatalf("seed usage: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`insert into refresh_tokens (token_hash, user_id, family_id, expires_at)
		 values ('h', $1, 'f', now() + interval '1 day')`, uidA); err != nil {
		t.Fatalf("seed refresh token: %v", err)
	}

	rec := do(t, mux, http.MethodDelete, "/api/account", ta,
		map[string]string{"confirm_email": "a@x.com"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rec.Code)
	}
	if cookies := rec.Result().Cookies(); len(cookies) == 0 {
		t.Fatalf("expected a cleared refresh cookie")
	}

	for _, tbl := range []string{"users", "conversations", "messages", "token_usage", "refresh_tokens"} {
		var n int
		if err := testPool.QueryRow(ctx,
			"select count(*) from "+tbl).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", tbl, err)
		}
		if n != 0 {
			t.Fatalf("%s: want 0 rows after delete, got %d", tbl, n)
		}
	}
}

func TestAccount_DeleteRequiresAuth(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	rec := do(t, mux, http.MethodDelete, "/api/account", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestAccount_DeleteRejectsWrongEmail(t *testing.T) {
	resetDB(t)
	mux := newTestMux(nil)
	ta, _ := signup(t, mux, "a@x.com")

	rec := do(t, mux, http.MethodDelete, "/api/account", ta,
		map[string]string{"confirm_email": "b@x.com"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}

	var n int
	if err := testPool.QueryRow(context.Background(),
		"select count(*) from users").Scan(&n); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if n != 1 {
		t.Fatalf("want user still present, got %d rows", n)
	}
}
