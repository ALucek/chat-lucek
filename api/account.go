package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	exportRatePerMin = 6
	exportRateBurst  = 6
)

// Account holds the self-serve account data handlers.
type Account struct {
	pool *pgxpool.Pool
}

type exportProfile struct {
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type exportMessage struct {
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	CreatedAt time.Time       `json:"created_at"`
	Trace     json.RawMessage `json:"trace,omitempty"`
}

type exportConversation struct {
	ID        int64           `json:"id"`
	Title     string          `json:"title"`
	Summary   *string         `json:"summary,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Messages  []exportMessage `json:"messages"`
}

type exportUsage struct {
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CreatedAt        time.Time `json:"created_at"`
}

type accountExport struct {
	ExportedAt    time.Time            `json:"exported_at"`
	Profile       exportProfile        `json:"profile"`
	Conversations []exportConversation `json:"conversations"`
	Usage         []exportUsage        `json:"usage"`
}

// Export writes the caller's full stored data as a JSON attachment.
func (a *Account) Export(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	ctx := r.Context()

	var out accountExport
	out.ExportedAt = time.Now().UTC()
	out.Conversations = []exportConversation{}
	out.Usage = []exportUsage{}

	if err := a.pool.QueryRow(ctx,
		`select email, created_at from users where id = $1`, userID).
		Scan(&out.Profile.Email, &out.Profile.CreatedAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load profile"})
		return
	}

	convRows, err := a.pool.Query(ctx,
		`select id, coalesce(title,''), summary, created_at, updated_at
		 from conversations where user_id = $1 order by id`, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load conversations"})
		return
	}
	for convRows.Next() {
		var c exportConversation
		if err := convRows.Scan(&c.ID, &c.Title, &c.Summary, &c.CreatedAt, &c.UpdatedAt); err != nil {
			convRows.Close()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		c.Messages = []exportMessage{}
		out.Conversations = append(out.Conversations, c)
	}
	convRows.Close()

	// Index after the slice is fully built so the pointers stay valid.
	byID := map[int64]*exportConversation{}
	for i := range out.Conversations {
		byID[out.Conversations[i].ID] = &out.Conversations[i]
	}

	msgRows, err := a.pool.Query(ctx,
		`select m.conversation_id, m.role, m.content, m.created_at, m.trace
		 from messages m join conversations c on c.id = m.conversation_id
		 where c.user_id = $1 order by m.conversation_id, m.id`, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load messages"})
		return
	}
	for msgRows.Next() {
		var cid int64
		var m exportMessage
		if err := msgRows.Scan(&cid, &m.Role, &m.Content, &m.CreatedAt, &m.Trace); err != nil {
			msgRows.Close()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		if c := byID[cid]; c != nil {
			c.Messages = append(c.Messages, m)
		}
	}
	msgRows.Close()

	usageRows, err := a.pool.Query(ctx,
		`select prompt_tokens, completion_tokens, created_at
		 from token_usage where user_id = $1 order by id`, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load usage"})
		return
	}
	for usageRows.Next() {
		var u exportUsage
		if err := usageRows.Scan(&u.PromptTokens, &u.CompletionTokens, &u.CreatedAt); err != nil {
			usageRows.Close()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		out.Usage = append(out.Usage, u)
	}
	usageRows.Close()

	filename := fmt.Sprintf("chat-lucek-export-%s.json", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ") // pretty-print so the download is readable
	if err := enc.Encode(out); err != nil {
		slog.ErrorContext(ctx, "export encode", "err", err)
	}
}

// Delete removes the account after the caller re-types their email.
func (a *Account) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	var body struct {
		ConfirmEmail string `json:"confirm_email"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	// Match on id AND email so the confirmation is enforced server-side.
	tag, err := a.pool.Exec(r.Context(),
		`delete from users where id = $1 and email = $2`,
		userID, normalizeEmail(body.ConfirmEmail))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not delete account"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email confirmation does not match"})
		return
	}
	http.SetCookie(w, refreshCookie("", -1)) // clear the refresh cookie
	w.WriteHeader(http.StatusNoContent)
}
