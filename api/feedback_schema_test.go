package main

import (
	"context"
	"testing"
)

func TestMigration_MessageFeedbackShape(t *testing.T) {
	resetDB(t)
	ctx := context.Background()

	var uid int64
	if err := testPool.QueryRow(ctx,
		`insert into users (email) values ('a@x.com') returning id`).Scan(&uid); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	var cid int64
	if err := testPool.QueryRow(ctx,
		`insert into conversations (user_id) values ($1) returning id`, uid).Scan(&cid); err != nil {
		t.Fatalf("seed conversation: %v", err)
	}
	var mid int64
	if err := testPool.QueryRow(ctx,
		`insert into messages (conversation_id, role, content, langsmith_run_id)
		 values ($1, 'assistant', 'hi', 'run-1') returning id`, cid).Scan(&mid); err != nil {
		t.Fatalf("seed message: %v", err)
	}

	if _, err := testPool.Exec(ctx,
		`insert into message_feedback (message_id, user_id, rating, comment, langsmith_feedback_id)
		 values ($1, $2, 1, 'nice', gen_random_uuid())`, mid, uid); err != nil {
		t.Fatalf("insert feedback: %v", err)
	}
	// Second row for the same (message, user) must violate the unique constraint.
	_, err := testPool.Exec(ctx,
		`insert into message_feedback (message_id, user_id, rating, langsmith_feedback_id)
		 values ($1, $2, -1, gen_random_uuid())`, mid, uid)
	if err == nil {
		t.Fatal("expected unique violation on (message_id, user_id)")
	}
}
