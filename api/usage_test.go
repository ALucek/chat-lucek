package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestRunsSince_CountsWithinWindow(t *testing.T) {
	resetDB(t)
	ctx := context.Background()

	var uid int64
	if err := testPool.QueryRow(ctx,
		`insert into users (google_sub, email) values ('sub:u@x.com', 'u@x.com') returning id`).
		Scan(&uid); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	if err := recordUsage(ctx, testPool, uid, tokenUsage{Prompt: 10, Completion: 5}); err != nil {
		t.Fatalf("record 1: %v", err)
	}
	if err := recordUsage(ctx, testPool, uid, tokenUsage{Prompt: 3, Completion: 2}); err != nil {
		t.Fatalf("record 2: %v", err)
	}

	// A row older than the window must not count.
	if _, err := testPool.Exec(ctx,
		`insert into token_usage (user_id, prompt_tokens, completion_tokens, created_at)
		 values ($1, 100, 100, now() - interval '25 hours')`, uid); err != nil {
		t.Fatalf("seed old: %v", err)
	}

	n, err := runsSince(ctx, testPool, uid, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("runsSince: %v", err)
	}
	if n != 2 {
		t.Fatalf("want 2 runs within window, got %d", n)
	}
}

func TestRunsSince_SurvivesConversationDelete(t *testing.T) {
	resetDB(t)
	ctx := context.Background()

	var uid int64
	if err := testPool.QueryRow(ctx,
		`insert into users (google_sub, email) values ('sub:u@x.com', 'u@x.com') returning id`).
		Scan(&uid); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	var cid int64
	if err := testPool.QueryRow(ctx,
		`insert into conversations (user_id) values ($1) returning id`, uid).Scan(&cid); err != nil {
		t.Fatalf("seed conversation: %v", err)
	}
	if err := recordUsage(ctx, testPool, uid, tokenUsage{Prompt: 7, Completion: 3}); err != nil {
		t.Fatalf("record: %v", err)
	}

	if _, err := testPool.Exec(ctx, `delete from conversations where id = $1`, cid); err != nil {
		t.Fatalf("delete conversation: %v", err)
	}

	n, err := runsSince(ctx, testPool, uid, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("runsSince: %v", err)
	}
	if n != 1 {
		t.Fatalf("usage must survive conversation delete; want 1, got %d", n)
	}
}

func TestUsage_Endpoint(t *testing.T) {
	resetDB(t)
	mux := newTestMuxBudget(nil, 20)
	ta, uid := signup(t, mux, "a@x.com")

	ctx := context.Background()
	if err := recordUsage(ctx, testPool, uid, tokenUsage{Prompt: 10, Completion: 5}); err != nil {
		t.Fatalf("record 1: %v", err)
	}
	if err := recordUsage(ctx, testPool, uid, tokenUsage{Prompt: 3, Completion: 2}); err != nil {
		t.Fatalf("record 2: %v", err)
	}
	// A row older than the 24h window must not count.
	if _, err := testPool.Exec(ctx,
		`insert into token_usage (user_id, prompt_tokens, completion_tokens, created_at)
		 values ($1, 100, 100, now() - interval '25 hours')`, uid); err != nil {
		t.Fatalf("seed old: %v", err)
	}

	rec := do(t, mux, http.MethodGet, "/api/usage", ta, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body)
	}
	var out struct {
		Used   int `json:"used"`
		Budget int `json:"budget"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Used != 2 {
		t.Fatalf("want used 2 runs, got %d", out.Used)
	}
	if out.Budget != 20 {
		t.Fatalf("want budget 20, got %d", out.Budget)
	}
}

func TestUsage_Endpoint_RequiresAuth(t *testing.T) {
	resetDB(t)
	mux := newTestMuxBudget(nil, 20)
	rec := do(t, mux, http.MethodGet, "/api/usage", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestSend_OwnerBypassesBudget(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusOK, textFrames("a", "hi"), endFrame)
	auth := &Auth{pool: testPool, secret: testSecret, verify: fakeGoogleVerifier(), exchange: fakeGoogleExchanger()}
	chat := &Chat{pool: testPool, agent: client, runsBudget: 1, ownerEmail: "owner@gmail.com"}
	mux := newMux(func(ctx context.Context) error { return Healthy(ctx, testPool) }, auth, chat, &Account{pool: testPool})

	// Owner, already over budget → still allowed.
	ownerTok, ownerID := signup(t, mux, "owner@gmail.com")
	if err := recordUsage(context.Background(), testPool, ownerID, tokenUsage{Prompt: 1, Completion: 1}); err != nil {
		t.Fatalf("seed owner usage: %v", err)
	}
	ownerConv := createConversation(t, mux, ownerTok)
	if rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", ownerConv), ownerTok,
		map[string]string{"content": "hi"}); rec.Code == http.StatusTooManyRequests {
		t.Fatal("owner should bypass the daily budget, got 429")
	}
	// The non-owner-over-budget → 429 path is covered by TestSend_OverRunBudget.
}
