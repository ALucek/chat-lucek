package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// seedAssistantMessage inserts an assistant message with a run id.
func seedAssistantMessage(t *testing.T, cid int64, runID string) int64 {
	t.Helper()
	var mid int64
	if err := testPool.QueryRow(context.Background(),
		`insert into messages (conversation_id, role, content, langsmith_run_id)
		 values ($1, 'assistant', 'answer', $2) returning id`, cid, runID).Scan(&mid); err != nil {
		t.Fatalf("seed message: %v", err)
	}
	return mid
}

func TestFeedback_UpsertAndMirror(t *testing.T) {
	resetDB(t)
	fm := &fakeMirror{}
	mux := newTestMuxFull(nil, testRunsBudget, fm)
	ta, uid := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	mid := seedAssistantMessage(t, cid, "run-1")

	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/messages/%d/feedback", mid), ta,
		map[string]any{"rating": 1, "comment": "great"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rec.Code)
	}

	var rating int
	var comment string
	if err := testPool.QueryRow(context.Background(),
		`select rating, coalesce(comment,'') from message_feedback where message_id=$1 and user_id=$2`,
		mid, uid).Scan(&rating, &comment); err != nil {
		t.Fatalf("query feedback: %v", err)
	}
	if rating != 1 || comment != "great" {
		t.Fatalf("got rating=%d comment=%q", rating, comment)
	}
	if len(fm.scores) != 1 || fm.scores[0].runID != "run-1" || fm.scores[0].score != 1.0 {
		t.Fatalf("score calls: %+v", fm.scores)
	}
	if len(fm.comments) != 1 || fm.comments[0].comment != "great" {
		t.Fatalf("comment calls: %+v", fm.comments)
	}
}

func TestFeedback_EmptyCommentDeletesNote(t *testing.T) {
	resetDB(t)
	fm := &fakeMirror{}
	mux := newTestMuxFull(nil, testRunsBudget, fm)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	mid := seedAssistantMessage(t, cid, "run-1")

	do(t, mux, http.MethodPost, fmt.Sprintf("/api/messages/%d/feedback", mid), ta,
		map[string]any{"rating": 1})
	if len(fm.comments) != 0 {
		t.Fatalf("no note should mean no comment upsert, got %+v", fm.comments)
	}
	if len(fm.deletes) != 1 {
		t.Fatalf("no note should reconcile with a delete, got %+v", fm.deletes)
	}
}

func TestFeedback_SwitchUpdatesSameRow(t *testing.T) {
	resetDB(t)
	mux := newTestMuxFull(nil, testRunsBudget, &fakeMirror{})
	ta, uid := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	mid := seedAssistantMessage(t, cid, "run-1")

	do(t, mux, http.MethodPost, fmt.Sprintf("/api/messages/%d/feedback", mid), ta, map[string]any{"rating": 1})
	do(t, mux, http.MethodPost, fmt.Sprintf("/api/messages/%d/feedback", mid), ta, map[string]any{"rating": -1})

	var count, rating int
	if err := testPool.QueryRow(context.Background(),
		`select count(*), max(rating) from message_feedback where message_id=$1 and user_id=$2`,
		mid, uid).Scan(&count, &rating); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 || rating != -1 {
		t.Fatalf("want single row rating -1, got count=%d rating=%d", count, rating)
	}
}

func TestFeedback_NotOwner(t *testing.T) {
	resetDB(t)
	mux := newTestMuxFull(nil, testRunsBudget, &fakeMirror{})
	ta, _ := signup(t, mux, "a@x.com")
	tb, _ := signup(t, mux, "b@x.com")
	cid := createConversation(t, mux, ta)
	mid := seedAssistantMessage(t, cid, "run-1")

	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/messages/%d/feedback", mid), tb,
		map[string]any{"rating": 1})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}

func TestFeedback_InvalidRating(t *testing.T) {
	resetDB(t)
	mux := newTestMuxFull(nil, testRunsBudget, &fakeMirror{})
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	mid := seedAssistantMessage(t, cid, "run-1")

	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/messages/%d/feedback", mid), ta,
		map[string]any{"rating": 0})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestFeedback_CommentTooLong(t *testing.T) {
	resetDB(t)
	mux := newTestMuxFull(nil, testRunsBudget, &fakeMirror{})
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	mid := seedAssistantMessage(t, cid, "run-1")

	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/messages/%d/feedback", mid), ta,
		map[string]any{"rating": 1, "comment": strings.Repeat("x", maxFeedbackChars+1)})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestMessages_IncludesCallerRating(t *testing.T) {
	resetDB(t)
	mux := newTestMuxFull(nil, testRunsBudget, &fakeMirror{})
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	mid := seedAssistantMessage(t, cid, "run-1")
	unrated := seedAssistantMessage(t, cid, "run-2")

	do(t, mux, http.MethodPost, fmt.Sprintf("/api/messages/%d/feedback", mid), ta,
		map[string]any{"rating": -1})

	body := do(t, mux, http.MethodGet,
		fmt.Sprintf("/api/conversations/%d/messages", cid), ta, nil).Body.Bytes()
	var msgs []struct {
		ID       int64 `json:"id"`
		Feedback *struct {
			Rating int `json:"rating"`
		} `json:"feedback"`
	}
	if err := json.Unmarshal(body, &msgs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var rated, plain bool
	for _, m := range msgs {
		if m.ID == mid {
			if m.Feedback == nil || m.Feedback.Rating != -1 {
				t.Fatalf("rated message feedback: %+v", m.Feedback)
			}
			rated = true
		}
		if m.ID == unrated {
			if m.Feedback != nil {
				t.Fatalf("unrated message should have no feedback, got %+v", m.Feedback)
			}
			plain = true
		}
	}
	if !rated || !plain {
		t.Fatalf("missing messages: rated=%v plain=%v", rated, plain)
	}
}

func TestFeedback_NoRunIDSkipsMirror(t *testing.T) {
	resetDB(t)
	fm := &fakeMirror{}
	mux := newTestMuxFull(nil, testRunsBudget, fm)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	var mid int64
	if err := testPool.QueryRow(context.Background(),
		`insert into messages (conversation_id, role, content) values ($1, 'assistant', 'answer') returning id`,
		cid).Scan(&mid); err != nil {
		t.Fatalf("seed: %v", err)
	}
	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/messages/%d/feedback", mid), ta,
		map[string]any{"rating": 1})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rec.Code)
	}
	if len(fm.scores) != 0 || len(fm.comments) != 0 || len(fm.deletes) != 0 {
		t.Fatalf("mirror should be skipped without a run id: %+v %+v %+v", fm.scores, fm.comments, fm.deletes)
	}
}
