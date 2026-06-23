package main

import (
	"context"
	"testing"
	"time"
)

func TestUsage_RecordAndSumWithinWindow(t *testing.T) {
	resetDB(t)
	ctx := context.Background()

	var uid int64
	if err := testPool.QueryRow(ctx,
		`insert into users (email, password_hash) values ('u@x.com', 'x') returning id`).
		Scan(&uid); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	if err := recordUsage(ctx, testPool, uid, tokenUsage{Prompt: 10, Completion: 5}); err != nil {
		t.Fatalf("record 1: %v", err)
	}
	if err := recordUsage(ctx, testPool, uid, tokenUsage{Prompt: 3, Completion: 2}); err != nil {
		t.Fatalf("record 2: %v", err)
	}

	// A row older than the window must be excluded.
	if _, err := testPool.Exec(ctx,
		`insert into token_usage (user_id, prompt_tokens, completion_tokens, created_at)
		 values ($1, 100, 100, now() - interval '25 hours')`, uid); err != nil {
		t.Fatalf("seed old: %v", err)
	}

	total, err := usageSince(ctx, testPool, uid, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("usageSince: %v", err)
	}
	if total != 20 {
		t.Fatalf("want 20 within window, got %d", total)
	}
}

func TestUsage_SurvivesConversationDelete(t *testing.T) {
	resetDB(t)
	ctx := context.Background()

	var uid int64
	if err := testPool.QueryRow(ctx,
		`insert into users (email, password_hash) values ('u@x.com', 'x') returning id`).
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

	total, err := usageSince(ctx, testPool, uid, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("usageSince: %v", err)
	}
	if total != 10 {
		t.Fatalf("usage must survive conversation delete; want 10, got %d", total)
	}
}
