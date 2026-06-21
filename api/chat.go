package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Chat holds the conversation/message handlers and their dependencies.
type Chat struct {
	pool *pgxpool.Pool
}

type conversation struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type message struct {
	ID        int64     `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

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

	// Ownership pre-check: distinguishes "not yours / missing" (404) from
	// "yours but empty" (200 []), which an empty result set alone cannot.
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
		`select id, role, content, created_at from messages
		 where conversation_id = $1 order by id`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load messages"})
		return
	}
	defer rows.Close()

	msgs := []message{}
	for rows.Next() {
		var m message
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		msgs = append(msgs, m)
	}
	writeJSON(w, http.StatusOK, msgs)
}
