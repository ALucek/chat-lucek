package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testPool *pgxpool.Pool

var testSecret = []byte("test-secret-at-least-32-bytes-long-xx")

var testUsageSecret = []byte("test-usage-secret-at-least-32-bytes-xx")

const testRunsBudget = 1_000_000

// TestMain starts one Postgres container, migrates, then runs all tests.
func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("chat"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("secret"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres: %v\n", err)
		os.Exit(1)
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "conn string: %v\n", err)
		os.Exit(1)
	}
	if err := runMigrations(dsn); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
	testPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pool: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()

	testPool.Close()
	_ = testcontainers.TerminateContainer(ctr)
	os.Exit(code)
}

// resetDB clears all app tables; call at the top of each integration test.
func resetDB(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		`truncate users, refresh_tokens, conversations, messages, token_usage, usage_marks restart identity cascade`)
	if err != nil {
		t.Fatalf("reset db: %v", err)
	}
}

// newTestMux builds the router against the test pool with a generous budget.
func newTestMux(client *agentClient) http.Handler {
	return newTestMuxBudget(client, testRunsBudget)
}

// newTestMuxBudget builds the router with an explicit daily run budget.
func newTestMuxBudget(client *agentClient, budget int) http.Handler {
	auth := &Auth{pool: testPool, secret: testSecret, verify: fakeGoogleVerifier(), exchange: fakeGoogleExchanger(), signupOpen: true}
	chat := &Chat{pool: testPool, agent: client, runsBudget: budget, usageSecret: testUsageSecret}
	account := &Account{pool: testPool}
	check := func(ctx context.Context) error { return Healthy(ctx, testPool) }
	return newMux(check, auth, chat, account)
}

// signup seeds a user directly and returns a freshly minted access token + id.
func signup(t *testing.T, _ http.Handler, email string) (token string, userID int64) {
	t.Helper()
	err := testPool.QueryRow(context.Background(),
		`insert into users (google_sub, email) values ($1, $2) returning id`,
		"sub:"+email, normalizeEmail(email)).Scan(&userID)
	if err != nil {
		t.Fatalf("seed user %s: %v", email, err)
	}
	token, err = mintAccessToken(testSecret, userID, time.Now())
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return token, userID
}

// seedMarks inserts n run marks for the account created by signup(email).
func seedMarks(t *testing.T, email string, n int) {
	t.Helper()
	sh := subjectHash(testUsageSecret, "sub:"+email)
	for i := 0; i < n; i++ {
		if _, err := testPool.Exec(context.Background(),
			`insert into usage_marks (subject_hash) values ($1)`, sh); err != nil {
			t.Fatalf("seed mark: %v", err)
		}
	}
}

// do sends a request through the mux; body is JSON-encoded when non-nil.
func do(t *testing.T, mux http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// TestHarness_Boots proves the container, migrations, and pool are wired up.
func TestHarness_Boots(t *testing.T) {
	resetDB(t)
	var n int
	if err := testPool.QueryRow(context.Background(), "select count(*) from users").Scan(&n); err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != 0 {
		t.Fatalf("want 0 users after reset, got %d", n)
	}
}
