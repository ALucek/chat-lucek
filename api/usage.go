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

// usageSince returns tokens (prompt+completion) a user spent since a time.
func usageSince(ctx context.Context, pool *pgxpool.Pool, userID int64, since time.Time) (int, error) {
	var total int
	err := pool.QueryRow(ctx,
		`select coalesce(sum(prompt_tokens + completion_tokens), 0)
		 from token_usage where user_id = $1 and created_at > $2`,
		userID, since).Scan(&total)
	return total, err
}

// Usage reports the caller's rolling-24h token usage against the daily budget.
func (c *Chat) Usage(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	used, err := usageSince(r.Context(), c.pool, userID, time.Now().Add(-24*time.Hour))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "usage check failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"used": used, "budget": c.tokenBudget})
}
