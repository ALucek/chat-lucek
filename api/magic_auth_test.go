package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// requestAndToken drives a magic request and returns the raw token.
func requestAndToken(t *testing.T, mux http.Handler, fm *fakeMailer, email string) string {
	t.Helper()
	do(t, mux, http.MethodPost, "/api/magic/request", "", map[string]string{"email": email})
	link, ok := fm.last(email)
	if !ok {
		t.Fatalf("no link captured for %s", email)
	}
	return link[strings.Index(link, "token=")+len("token="):]
}

func magicMux() (http.Handler, *fakeMailer) {
	fm := newFakeMailer()
	auth := &Auth{pool: testPool, secret: testSecret, signupOpen: true, mailer: fm, linkBase: "http://localhost:3000"}
	chat := &Chat{pool: testPool, runsBudget: testRunsBudget, usageSecret: testUsageSecret}
	mux := newMux(func(ctx context.Context) error { return Healthy(ctx, testPool) }, auth, chat, &Account{pool: testPool})
	return mux, fm
}

func TestMagicRequest_SendsLinkAndReturns200(t *testing.T) {
	resetDB(t)
	mux, fm := magicMux()
	rec := do(t, mux, http.MethodPost, "/api/magic/request", "", map[string]string{"email": "new@gmail.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body)
	}
	if _, ok := fm.last("new@gmail.com"); !ok {
		t.Fatal("expected a link to be sent")
	}
}

func TestMagicRequest_RejectsInvalidEmail(t *testing.T) {
	resetDB(t)
	mux, _ := magicMux()
	rec := do(t, mux, http.MethodPost, "/api/magic/request", "", map[string]string{"email": "not-an-email"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestMagicRequest_SkipsDisposableStill200(t *testing.T) {
	resetDB(t)
	mux, fm := magicMux()
	rec := do(t, mux, http.MethodPost, "/api/magic/request", "", map[string]string{"email": "x@mailinator.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if _, ok := fm.last("x@mailinator.com"); ok {
		t.Fatal("disposable domain should not be sent a link")
	}
}

func TestMagicRequest_ClosedSignupSkipsNewEmailStill200(t *testing.T) {
	resetDB(t)
	fm := newFakeMailer()
	auth := &Auth{pool: testPool, secret: testSecret, signupOpen: false, mailer: fm, linkBase: "http://localhost:3000"}
	chat := &Chat{pool: testPool, runsBudget: testRunsBudget, usageSecret: testUsageSecret}
	mux := newMux(func(ctx context.Context) error { return Healthy(ctx, testPool) }, auth, chat, &Account{pool: testPool})
	rec := do(t, mux, http.MethodPost, "/api/magic/request", "", map[string]string{"email": "stranger@gmail.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if _, ok := fm.last("stranger@gmail.com"); ok {
		t.Fatal("closed signup should not send a link to a new email")
	}
}

func TestMagicRequest_RateLimited(t *testing.T) {
	resetDB(t)
	mux, _ := magicMux()
	var last int
	for i := 0; i < magicRateBurst+2; i++ {
		last = do(t, mux, http.MethodPost, "/api/magic/request", "",
			map[string]string{"email": "flood@gmail.com"}).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("want 429 after burst, got %d", last)
	}
}

func TestMagicVerify_ValidTokenIssuesSession(t *testing.T) {
	resetDB(t)
	mux, fm := magicMux()
	tok := requestAndToken(t, mux, fm, "alice@gmail.com")
	rec := do(t, mux, http.MethodPost, "/api/magic/verify", "", map[string]string{"token": tok})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body)
	}
	var n int
	testPool.QueryRow(context.Background(),
		`select count(*) from users where email = 'alice@gmail.com'`).Scan(&n)
	if n != 1 {
		t.Fatalf("want 1 user, got %d", n)
	}
}

func TestMagicVerify_SingleUse(t *testing.T) {
	resetDB(t)
	mux, fm := magicMux()
	tok := requestAndToken(t, mux, fm, "bob@gmail.com")
	do(t, mux, http.MethodPost, "/api/magic/verify", "", map[string]string{"token": tok})
	rec := do(t, mux, http.MethodPost, "/api/magic/verify", "", map[string]string{"token": tok})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("second use want 401, got %d", rec.Code)
	}
}

func TestMagicVerify_BadToken(t *testing.T) {
	resetDB(t)
	mux, _ := magicMux()
	rec := do(t, mux, http.MethodPost, "/api/magic/verify", "", map[string]string{"token": "nope"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestMagicVerify_ClosedSignupRejectsNewEmail(t *testing.T) {
	resetDB(t)
	fm := newFakeMailer()
	// Seed the link with signups open, then verify against a closed-signup mux.
	openAuth := &Auth{pool: testPool, secret: testSecret, signupOpen: true, mailer: fm, linkBase: "http://localhost:3000"}
	chat := &Chat{pool: testPool, runsBudget: testRunsBudget, usageSecret: testUsageSecret}
	openMux := newMux(func(ctx context.Context) error { return Healthy(ctx, testPool) }, openAuth, chat, &Account{pool: testPool})
	tok := requestAndToken(t, openMux, fm, "late@gmail.com")

	closedAuth := &Auth{pool: testPool, secret: testSecret, signupOpen: false, mailer: fm, linkBase: "http://localhost:3000"}
	closedMux := newMux(func(ctx context.Context) error { return Healthy(ctx, testPool) }, closedAuth, chat, &Account{pool: testPool})
	rec := do(t, closedMux, http.MethodPost, "/api/magic/verify", "", map[string]string{"token": tok})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("new email under closed signup want 403, got %d", rec.Code)
	}
}

func TestMagicLatest_AbsentUnderRealMailer(t *testing.T) {
	resetDB(t)
	auth := &Auth{pool: testPool, secret: testSecret, signupOpen: true, mailer: &resendMailer{}, linkBase: "https://x"}
	chat := &Chat{pool: testPool, runsBudget: testRunsBudget, usageSecret: testUsageSecret}
	mux := newMux(func(ctx context.Context) error { return Healthy(ctx, testPool) }, auth, chat, &Account{pool: testPool})
	rec := do(t, mux, http.MethodGet, "/api/magic/latest?email=a@b.com", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 under real mailer, got %d", rec.Code)
	}
}
