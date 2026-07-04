package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

// runsSince counts a user's completed runs (one usage row each) since a time.
func runsSince(ctx context.Context, pool *pgxpool.Pool, userID int64, since time.Time) (int, error) {
	var n int
	err := pool.QueryRow(ctx,
		`select count(*) from token_usage where user_id = $1 and created_at > $2`,
		userID, since).Scan(&n)
	return n, err
}

// Usage reports the caller's rolling-24h run count against the daily budget.
func (c *Chat) Usage(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	used, err := runsSince(r.Context(), c.pool, userID, time.Now().Add(-24*time.Hour))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "usage check failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"used": used, "budget": c.runsBudget})
}
