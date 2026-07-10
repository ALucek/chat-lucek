package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestCountMarks_ExcludesOldAndOtherSubjects(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	sh := subjectHash(testUsageSecret, canonicalizeEmail("a@x.com"))
	other := subjectHash(testUsageSecret, canonicalizeEmail("b@x.com"))
	if _, err := testPool.Exec(ctx,
		`insert into usage_marks (subject_hash) values ($1), ($1)`, sh); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`insert into usage_marks (subject_hash, created_at) values ($1, now() - interval '25 hours')`, sh); err != nil {
		t.Fatalf("seed old: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`insert into usage_marks (subject_hash) values ($1)`, other); err != nil {
		t.Fatalf("seed other: %v", err)
	}

	n, err := countMarks(ctx, testPool, sh, time.Now().Add(-budgetWindow))
	if err != nil {
		t.Fatalf("countMarks: %v", err)
	}
	if n != 2 {
		t.Fatalf("want 2 in-window marks for the subject, got %d", n)
	}
}

func TestBudget_SurvivesAccountDeleteAndResignup(t *testing.T) {
	resetDB(t)
	mux := newTestMuxBudget(nil, 1) // budget of one run
	_, _ = signup(t, mux, "churn@x.com")
	seedMarks(t, "churn@x.com", 1) // already at the cap

	// Hard-delete the account; usage_marks has no FK, so it survives.
	if _, err := testPool.Exec(context.Background(),
		`delete from users where email = 'churn@x.com'`); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	// Re-signup: new user id, same email, so the same subject_hash.
	ta, _ := signup(t, mux, "churn@x.com")
	conv := createConversation(t, mux, ta)
	rec := do(t, mux, http.MethodPost,
		fmt.Sprintf("/api/conversations/%d/messages", conv), ta,
		map[string]string{"content": "hi"})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("budget must survive delete+resignup; want 429, got %d", rec.Code)
	}
}

func TestUsage_Endpoint(t *testing.T) {
	resetDB(t)
	mux := newTestMuxBudget(nil, 20)
	ta, _ := signup(t, mux, "a@x.com")

	ctx := context.Background()
	seedMarks(t, "a@x.com", 2)
	// A mark older than the 24h window must not count.
	if _, err := testPool.Exec(ctx,
		`insert into usage_marks (subject_hash, created_at) values ($1, now() - interval '25 hours')`,
		subjectHash(testUsageSecret, canonicalizeEmail("a@x.com"))); err != nil {
		t.Fatalf("seed old: %v", err)
	}

	rec := do(t, mux, http.MethodGet, "/api/usage", ta, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body)
	}
	var out struct {
		Used   int `json:"used"`
		Budget int `json:"budget"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Used != 2 {
		t.Fatalf("want used 2 runs, got %d", out.Used)
	}
	if out.Budget != 20 {
		t.Fatalf("want budget 20, got %d", out.Budget)
	}
}

func TestUsage_Endpoint_RequiresAuth(t *testing.T) {
	resetDB(t)
	mux := newTestMuxBudget(nil, 20)
	rec := do(t, mux, http.MethodGet, "/api/usage", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestSend_OwnerBypassesBudget(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusOK, textFrames("a", "hi"), endFrame)
	auth := &Auth{pool: testPool, secret: testSecret, verify: fakeGoogleVerifier(), exchange: fakeGoogleExchanger()}
	chat := &Chat{pool: testPool, agent: client, runsBudget: 1, ownerEmail: "owner@gmail.com", usageSecret: testUsageSecret}
	mux := newMux(func(ctx context.Context) error { return Healthy(ctx, testPool) }, auth, chat, &Account{pool: testPool})

	// Owner, already over budget → still allowed.
	ownerTok, _ := signup(t, mux, "owner@gmail.com")
	seedMarks(t, "owner@gmail.com", 1)
	ownerConv := createConversation(t, mux, ownerTok)
	if rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", ownerConv), ownerTok,
		map[string]string{"content": "hi"}); rec.Code == http.StatusTooManyRequests {
		t.Fatal("owner should bypass the daily budget, got 429")
	}
	// The non-owner-over-budget → 429 path is covered by TestSend_OverRunBudget.
}
