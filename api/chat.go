package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Chat holds the conversation/message handlers and their dependencies.
type Chat struct {
	pool       *pgxpool.Pool
	agent      *agentClient
	runsBudget int
	ownerEmail string
}

type conversation struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type message struct {
	ID        int64           `json:"id"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	CreatedAt time.Time       `json:"created_at"`
	Trace     json.RawMessage `json:"trace,omitempty"`
}

// traceNode is one node in a run's event log (persisted and streamed).
type traceNode struct {
	ID       string          `json:"id"`
	ParentID *string         `json:"parent_id"`
	Type     string          `json:"type"`
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
	Output   json.RawMessage `json:"output,omitempty"`
	Text     string          `json:"text,omitempty"`
}

// messageTrace is a run's ordered event log persisted on a reply.
type messageTrace struct {
	Version int         `json:"version"`
	Nodes   []traceNode `json:"nodes"`
}

// nonTrivial reports whether a run has anything beyond top-level answer text.
func nonTrivial(nodes []*traceNode) bool {
	for _, n := range nodes {
		if !(n.Type == "text" && n.ParentID == nil) {
			return true
		}
	}
	return false
}

const maxMessageChars = 8000

// List returns the caller's conversations, newest activity first.
func (c *Chat) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	rows, err := c.pool.Query(r.Context(),
		`select id, coalesce(title,''), created_at, updated_at
		 from conversations where user_id = $1 order by updated_at desc`, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not list conversations"})
		return
	}
	defer rows.Close()

	list := []conversation{}
	for rows.Next() {
		var cv conversation
		if err := rows.Scan(&cv.ID, &cv.Title, &cv.CreatedAt, &cv.UpdatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		list = append(list, cv)
	}
	writeJSON(w, http.StatusOK, list)
}

// Create makes a new (untitled) conversation for the caller.
func (c *Chat) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	var cv conversation
	err := c.pool.QueryRow(r.Context(),
		`insert into conversations (user_id) values ($1)
		 returning id, coalesce(title,''), created_at, updated_at`, userID).
		Scan(&cv.ID, &cv.Title, &cv.CreatedAt, &cv.UpdatedAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create conversation"})
		return
	}
	writeJSON(w, http.StatusCreated, cv)
}

// conversationID parses the {id} path value.
func conversationID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// Messages returns one conversation's messages, oldest first.
func (c *Chat) Messages(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id, err := conversationID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid conversation id"})
		return
	}

	// Ownership pre-check: "not yours" (404) vs "yours but empty" (200 []).
	var owned bool
	if err := c.pool.QueryRow(r.Context(),
		`select exists(select 1 from conversations where id = $1 and user_id = $2)`,
		id, userID).Scan(&owned); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return
	}
	if !owned {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}

	rows, err := c.pool.Query(r.Context(),
		`select id, role, content, created_at, trace from messages
		 where conversation_id = $1 order by id`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load messages"})
		return
	}
	defer rows.Close()

	msgs := []message{}
	for rows.Next() {
		var m message
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.CreatedAt, &m.Trace); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		msgs = append(msgs, m)
	}
	writeJSON(w, http.StatusOK, msgs)
}

// Rename sets a conversation's title (scoped to the caller). 204 on success.
func (c *Chat) Rename(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id, err := conversationID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid conversation id"})
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title required"})
		return
	}
	// No updated_at bump: renaming is not new activity.
	tag, err := c.pool.Exec(r.Context(),
		`update conversations set title = $1 where id = $2 and user_id = $3`,
		body.Title, id, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	if tag.RowsAffected() == 0 { // not owned or missing
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Delete removes a conversation (caller-scoped); messages cascade. 204.
func (c *Chat) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id, err := conversationID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid conversation id"})
		return
	}
	tag, err := c.pool.Exec(r.Context(),
		`delete from conversations where id = $1 and user_id = $2`, id, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Send streams the assistant's reply to a new user message over SSE.
func (c *Chat) Send(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id, err := conversationID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid conversation id"})
		return
	}

	// Ownership pre-check (same as Messages): 404 if not the caller's.
	var owned bool
	if err := c.pool.QueryRow(r.Context(),
		`select exists(select 1 from conversations where id = $1 and user_id = $2)`,
		id, userID).Scan(&owned); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return
	}
	if !owned {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}

	var body struct {
		Content string `json:"content"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content required"})
		return
	}
	if utf8.RuneCountInString(body.Content) > maxMessageChars {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message too long (max 8000 characters)"})
		return
	}

	owner := false
	if c.ownerEmail != "" {
		var email string
		if c.pool.QueryRow(r.Context(), `select email from users where id = $1`, userID).Scan(&email) == nil {
			owner = email == c.ownerEmail
		}
	}
	if !owner {
		used, err := runsSince(r.Context(), c.pool, userID, time.Now().Add(-24*time.Hour))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "usage check failed"})
			return
		}
		if used >= c.runsBudget {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "daily run limit exceeded"})
			return
		}
	}

	// Persist the user message first, so it survives a failed model call.
	if _, err := c.pool.Exec(r.Context(),
		`insert into messages (conversation_id, role, content) values ($1, 'user', $2)`,
		id, body.Content); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save message"})
		return
	}

	// Build the request from full history; the agent owns the system prompt.
	msgs := []llmMessage{}
	rows, err := c.pool.Query(r.Context(),
		`select role, content from messages where conversation_id = $1 order by id`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load history"})
		return
	}
	for rows.Next() {
		var m llmMessage
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			rows.Close()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		msgs = append(msgs, m)
	}
	rows.Close() // free the pooled connection before the (long) stream

	firstMessage := len(msgs) == 1 // just the inserted user message

	// Commit to the stream: from here, failures are reported as SSE events.
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	var reply strings.Builder
	nodes := []*traceNode{}
	byID := map[string]*traceNode{}
	usage, err := c.agent.run(r.Context(), msgs, strconv.FormatInt(id, 10), runHandlers{
		onNode: func(f nodeFrame) {
			n := &traceNode{ID: f.ID, ParentID: f.ParentID, Type: f.Type, Name: f.Name, Input: f.Input}
			nodes = append(nodes, n)
			byID[f.ID] = n
			writeSSE(w, "node", f)
			flusher.Flush()
		},
		onDelta: func(id, text string) {
			if n := byID[id]; n != nil {
				n.Text += text
				if n.Type == "text" && n.ParentID == nil {
					reply.WriteString(text)
				}
			}
			writeSSE(w, "delta", map[string]string{"id": id, "text": text})
			flusher.Flush()
		},
		onNodeEnd: func(id string, output json.RawMessage) {
			if n := byID[id]; n != nil {
				n.Output = output
			}
			writeSSE(w, "node_end", map[string]any{"id": id, "output": output})
			flusher.Flush()
		},
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "stream", "err", err)
		writeSSE(w, "error", map[string]string{"error": "stream failed"})
		flusher.Flush()
		return
	}

	// Persist the complete reply and bump activity time.
	var traceJSON []byte
	if nonTrivial(nodes) {
		flat := make([]traceNode, len(nodes))
		for i, n := range nodes {
			flat[i] = *n
		}
		traceJSON, _ = json.Marshal(messageTrace{Version: 2, Nodes: flat})
	}
	var msgID int64
	if err := c.pool.QueryRow(r.Context(),
		`insert into messages (conversation_id, role, content, trace) values ($1, 'assistant', $2, $3) returning id`,
		id, reply.String(), traceJSON).Scan(&msgID); err != nil {
		slog.ErrorContext(r.Context(), "save reply", "err", err)
		writeSSE(w, "error", map[string]string{"error": "could not save reply"})
		flusher.Flush()
		return
	}
	_, _ = c.pool.Exec(r.Context(),
		`update conversations set updated_at = now() where id = $1`, id)

	if err := recordUsage(r.Context(), c.pool, userID, usage); err != nil {
		slog.ErrorContext(r.Context(), "record usage", "err", err)
	}
	writeSSE(w, "done", map[string]int64{"message_id": msgID})
	flusher.Flush()

	// On the first message, name the conversation from its opening words.
	if firstMessage {
		title := firstWords(body.Content, 5)
		_, _ = c.pool.Exec(r.Context(),
			`update conversations set title = $1 where id = $2`, title, id)
		writeSSE(w, "title", map[string]string{"title": title})
		flusher.Flush()
	}
}

// writeSSE writes one SSE frame; data is JSON so each frame is a JSON object.
func writeSSE(w http.ResponseWriter, event string, data any) {
	payload, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
}

// firstWords returns the first n whitespace-separated words of s.
func firstWords(s string, n int) string {
	words := strings.Fields(s)
	if len(words) > n {
		words = words[:n]
	}
	return strings.Join(words, " ")
}
