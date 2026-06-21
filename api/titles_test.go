package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestSend_FirstMessageSetsTitle(t *testing.T) {
	resetDB(t)
	client := fakeOpenRouter(t, http.StatusOK, deltaFrame("ok"), "data: [DONE]\n\n")
	mux := newTestMux(client)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)

	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "Plan a trip to Japan next year please"})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: title") || !strings.Contains(body, `"title":"Plan a trip to Japan"`) {
		t.Fatalf("missing title event: %s", body)
	}

	var title string
	if err := testPool.QueryRow(context.Background(),
		`select coalesce(title,'') from conversations where id=$1`, cid).Scan(&title); err != nil {
		t.Fatalf("query title: %v", err)
	}
	if title != "Plan a trip to Japan" {
		t.Fatalf("want title %q, got %q", "Plan a trip to Japan", title)
	}
}

func TestSend_SecondMessageDoesNotRetitle(t *testing.T) {
	resetDB(t)
	client := fakeOpenRouter(t, http.StatusOK, deltaFrame("ok"), "data: [DONE]\n\n")
	mux := newTestMux(client)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	path := fmt.Sprintf("/api/conversations/%d/messages", cid)

	// First message sets the title.
	do(t, mux, http.MethodPost, path, ta, map[string]string{"content": "alpha beta gamma delta epsilon zeta"})

	// Second message must not emit a title event or change the stored title.
	rec := do(t, mux, http.MethodPost, path, ta, map[string]string{"content": "another message entirely"})
	if strings.Contains(rec.Body.String(), "event: title") {
		t.Fatalf("second message should not emit a title event: %s", rec.Body)
	}
	var title string
	testPool.QueryRow(context.Background(),
		`select coalesce(title,'') from conversations where id=$1`, cid).Scan(&title)
	if title != "alpha beta gamma delta epsilon" {
		t.Fatalf("title changed on second message: %q", title)
	}
}
