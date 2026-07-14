package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Chat holds the conversation/message handlers and their dependencies.
type Chat struct {
	pool        *pgxpool.Pool
	agent       *agentClient
	runsBudget  int
	ownerEmail  string
	usageSecret []byte
	mirror      feedbackMirror
	devHost     string // requests to this host route to the candidate agent
}

type conversation struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Agentz canned-checks the api->agent path; dev host only (404 elsewhere).
func (c *Chat) Agentz(w http.ResponseWriter, r *http.Request) {
	if c.devHost == "" || r.Host != c.devHost {
		http.NotFound(w, r)
		return
	}
	gotText := false
	usage, err := c.agent.run(r.Context(),
		[]llmMessage{{Role: "user", Content: "What is your name?"}}, "", true,
		runHandlers{onDelta: func(_, text string) {
			if text != "" {
				gotText = true
			}
		}})
	if err != nil || !gotText || usage.Completion == 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "agent unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type message struct {
	ID        int64                `json:"id"`
	Role      string               `json:"role"`
	Content   string               `json:"content"`
	CreatedAt time.Time            `json:"created_at"`
	Trace     json.RawMessage      `json:"trace,omitempty"`
	Feedback  *messageFeedbackView `json:"feedback,omitempty"`
}

// messageFeedbackView is the caller's own rating on a message, for rehydration.
type messageFeedbackView struct {
	Rating int `json:"rating"`
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
const maxFeedbackChars = 2000

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

// ownsConversation writes the 404/500 itself; callers return when it's false.
func (c *Chat) ownsConversation(w http.ResponseWriter, r *http.Request, id, userID int64) bool {
	var owned bool
	if err := c.pool.QueryRow(r.Context(),
		`select exists(select 1 from conversations where id = $1 and user_id = $2)`,
		id, userID).Scan(&owned); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return false
	}
	if !owned {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return false
	}
	return true
}

// messageRunID returns the message's run id, or ok=false if not the caller's.
func (c *Chat) messageRunID(w http.ResponseWriter, r *http.Request, msgID, userID int64) (runID *string, ok bool) {
	err := c.pool.QueryRow(r.Context(),
		`select m.langsmith_run_id from messages m
		 join conversations conv on conv.id = m.conversation_id
		 where m.id = $1 and conv.user_id = $2`, msgID, userID).Scan(&runID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "message not found"})
		return nil, false
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return nil, false
	}
	return runID, true
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
	if !c.ownsConversation(w, r, id, userID) {
		return
	}

	rows, err := c.pool.Query(r.Context(),
		`select m.id, m.role, m.content, m.created_at, m.trace, mf.rating
		 from messages m
		 left join message_feedback mf on mf.message_id = m.id and mf.user_id = $2
		 where m.conversation_id = $1 order by m.id`, id, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load messages"})
		return
	}
	defer rows.Close()

	msgs := []message{}
	for rows.Next() {
		var m message
		var rating *int
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.CreatedAt, &m.Trace, &rating); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		if rating != nil {
			m.Feedback = &messageFeedbackView{Rating: *rating}
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
	if !c.ownsConversation(w, r, id, userID) {
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

	var email string
	if err := c.pool.QueryRow(r.Context(),
		`select email from users where id = $1`, userID).Scan(&email); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "usage check failed"})
		return
	}
	subject := subjectHash(c.usageSecret, canonicalizeEmail(email))
	owner := c.ownerEmail != "" && email == c.ownerEmail
	if !owner {
		used, err := countMarks(r.Context(), c.pool, subject, time.Now().Add(-budgetWindow))
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
	var runID string
	nodes := []*traceNode{}
	byID := map[string]*traceNode{}
	dev := c.devHost != "" && r.Host == c.devHost
	usage, err := c.agent.run(r.Context(), msgs, strconv.FormatInt(id, 10), dev, runHandlers{
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
		onMeta: func(rid string) { runID = rid },
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
	var runIDArg any
	if runID != "" {
		runIDArg = runID
	}
	var msgID int64
	if err := c.pool.QueryRow(r.Context(),
		`insert into messages (conversation_id, role, content, trace, langsmith_run_id) values ($1, 'assistant', $2, $3, $4) returning id`,
		id, reply.String(), traceJSON, runIDArg).Scan(&msgID); err != nil {
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
	if err := recordMark(r.Context(), c.pool, subject); err != nil {
		slog.ErrorContext(r.Context(), "record mark", "err", err)
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

// Feedback upserts the caller's rating and note, then mirrors to LangSmith.
func (c *Chat) Feedback(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	msgID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid message id"})
		return
	}
	var body struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Rating != -1 && body.Rating != 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "rating must be -1 or 1"})
		return
	}
	if utf8.RuneCountInString(body.Comment) > maxFeedbackChars {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "comment too long (max 2000 characters)"})
		return
	}

	// Ownership: the message must belong to one of the caller's conversations.
	runID, ok := c.messageRunID(w, r, msgID, userID)
	if !ok {
		return
	}

	feedbackID := uuid.NewString()
	var comment any
	if body.Comment != "" {
		comment = body.Comment
	}
	var savedFeedbackID string
	if err := c.pool.QueryRow(r.Context(),
		`insert into message_feedback (message_id, user_id, rating, comment, langsmith_feedback_id)
		 values ($1, $2, $3, $4, $5)
		 on conflict (message_id, user_id)
		 do update set rating = excluded.rating, comment = excluded.comment, updated_at = now()
		 returning langsmith_feedback_id`,
		msgID, userID, body.Rating, comment, feedbackID).Scan(&savedFeedbackID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "feedback failed"})
		return
	}

	if runID != nil && *runID != "" && c.mirror != nil && c.mirror.enabled() {
		score := 0.0
		if body.Rating == 1 {
			score = 1.0
		}
		c.mirror.upsertScore(savedFeedbackID, *runID, score)
		// The note rides its own key, with a stable id derived from the score's.
		commentID := commentFeedbackID(savedFeedbackID)
		if body.Comment != "" {
			c.mirror.upsertComment(commentID, *runID, body.Comment)
		} else {
			c.mirror.deleteFeedback(commentID)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearFeedback removes the caller's rating and note, and scrubs LangSmith.
func (c *Chat) ClearFeedback(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	msgID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid message id"})
		return
	}

	// Ownership: the message must belong to one of the caller's conversations.
	if _, ok := c.messageRunID(w, r, msgID, userID); !ok {
		return
	}

	var savedFeedbackID string
	err = c.pool.QueryRow(r.Context(),
		`delete from message_feedback where message_id = $1 and user_id = $2
		 returning langsmith_feedback_id`, msgID, userID).Scan(&savedFeedbackID)
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNoContent) // nothing to clear; idempotent
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "clear failed"})
		return
	}

	if c.mirror != nil && c.mirror.enabled() {
		c.mirror.deleteFeedback(savedFeedbackID)
		c.mirror.deleteFeedback(commentFeedbackID(savedFeedbackID))
	}
	w.WriteHeader(http.StatusNoContent)
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
