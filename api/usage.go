package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const budgetWindow = 24 * time.Hour

// tokenUsage is the prompt/completion token count for one completed LLM call.
type tokenUsage struct {
	Prompt     int
	Completion int
}

// recordUsage appends one row to the token-usage ledger.
func recordUsage(ctx context.Context, pool *pgxpool.Pool, userID int64, u tokenUsage) error {
	_, err := pool.Exec(ctx,
		`insert into token_usage (user_id, prompt_tokens, completion_tokens) values ($1, $2, $3)`,
		userID, u.Prompt, u.Completion)
	return err
}

// subjectHash is a stable per-user key that survives account deletion.
func subjectHash(secret []byte, googleSub string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(googleSub))
	return mac.Sum(nil)
}

// countMarks counts a subject's run marks since a time.
func countMarks(ctx context.Context, pool *pgxpool.Pool, subject []byte, since time.Time) (int, error) {
	var n int
	err := pool.QueryRow(ctx,
		`select count(*) from usage_marks where subject_hash = $1 and created_at > $2`,
		subject, since).Scan(&n)
	return n, err
}

// recordMark logs one run for a subject and drops that subject's stale marks.
func recordMark(ctx context.Context, pool *pgxpool.Pool, subject []byte) error {
	if _, err := pool.Exec(ctx,
		`insert into usage_marks (subject_hash) values ($1)`, subject); err != nil {
		return err
	}
	_, _ = pool.Exec(ctx,
		`delete from usage_marks where subject_hash = $1 and created_at < $2`,
		subject, time.Now().Add(-budgetWindow))
	return nil
}

// runsUsed counts a user's marks in the window, resolving their subject hash.
func runsUsed(ctx context.Context, pool *pgxpool.Pool, secret []byte, userID int64, since time.Time) (int, error) {
	var email string
	if err := pool.QueryRow(ctx,
		`select email from users where id = $1`, userID).Scan(&email); err != nil {
		return 0, err
	}
	return countMarks(ctx, pool, subjectHash(secret, canonicalizeEmail(email)), since)
}

// Usage reports the caller's rolling-24h run count against the daily budget.
func (c *Chat) Usage(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	used, err := runsUsed(r.Context(), c.pool, c.usageSecret, userID, time.Now().Add(-budgetWindow))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "usage check failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"used": used, "budget": c.runsBudget})
}
