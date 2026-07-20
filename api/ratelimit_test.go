package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func TestLimiter_AllowsBurstThenBlocks(t *testing.T) {
	now := time.Now()
	l := &limiter{entries: map[string]*entry{}, rate: 1, burst: 3, now: fixedClock(now)}
	for i := 0; i < 3; i++ {
		if ok, _ := l.allow("k"); !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	ok, retry := l.allow("k")
	if ok {
		t.Fatal("4th request should be blocked")
	}
	if retry <= 0 {
		t.Fatalf("want positive Retry-After, got %v", retry)
	}
}

func TestLimiter_RefillsOverTime(t *testing.T) {
	now := time.Now()
	clock := now
	l := &limiter{entries: map[string]*entry{}, rate: 1, burst: 1, now: func() time.Time { return clock }}
	if ok, _ := l.allow("k"); !ok {
		t.Fatal("first should be allowed")
	}
	if ok, _ := l.allow("k"); ok {
		t.Fatal("second should be blocked (bucket empty)")
	}
	clock = now.Add(2 * time.Second) // refills past 1 token, capped at burst
	if ok, _ := l.allow("k"); !ok {
		t.Fatal("should be allowed after refill")
	}
}

func TestLimiter_PerKeyIsolation(t *testing.T) {
	now := time.Now()
	l := &limiter{entries: map[string]*entry{}, rate: 1, burst: 1, now: fixedClock(now)}
	l.allow("a")
	if ok, _ := l.allow("b"); !ok {
		t.Fatal("key b has its own bucket")
	}
}

func TestLimiter_EvictsIdleKeys(t *testing.T) {
	now := time.Now()
	clock := now
	l := &limiter{
		entries: map[string]*entry{}, rate: 1, burst: 1,
		now: func() time.Time { return clock }, idleTTL: time.Minute,
	}
	l.allow("k")
	clock = now.Add(2 * time.Minute)
	l.evict()
	if len(l.entries) != 0 {
		t.Fatalf("idle key should be evicted, have %d", len(l.entries))
	}
}

func TestLimiter_Middleware429WithRetryAfter(t *testing.T) {
	now := time.Now()
	l := &limiter{entries: map[string]*entry{}, rate: 1, burst: 1, now: fixedClock(now)}
	h := rateLimit(l, func(*http.Request) string { return "k" })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))

	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec1.Code != http.StatusOK {
		t.Fatalf("first want 200, got %d", rec1.Code)
	}
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second want 429, got %d", rec2.Code)
	}
	if rec2.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header")
	}
}

func newTestRedisLimiter(t *testing.T, perMin int, clock *time.Time) (*redisLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return &redisLimiter{
		rdb: rdb, prefix: "rl:test:", window: time.Minute, limit: perMin,
		timeout: time.Second, now: func() time.Time { return *clock },
		fallback: newLimiter(perMin, perMin),
	}, mr
}

func TestRedisLimiter_AllowsThenBlocks(t *testing.T) {
	now := time.Now()
	rl, _ := newTestRedisLimiter(t, 3, &now)
	for i := 0; i < 3; i++ {
		if ok, _ := rl.allow("k"); !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	ok, retry := rl.allow("k")
	if ok {
		t.Fatal("4th request should be blocked")
	}
	if retry <= 0 {
		t.Fatalf("want positive Retry-After, got %v", retry)
	}
}

func TestRedisLimiter_WindowSlides(t *testing.T) {
	now := time.Now()
	rl, _ := newTestRedisLimiter(t, 1, &now)
	if ok, _ := rl.allow("k"); !ok {
		t.Fatal("first should be allowed")
	}
	if ok, _ := rl.allow("k"); ok {
		t.Fatal("second should be blocked")
	}
	now = now.Add(61 * time.Second)
	if ok, _ := rl.allow("k"); !ok {
		t.Fatal("should be allowed after the window slides")
	}
}

func TestRedisLimiter_SharesBudgetAcrossInstances(t *testing.T) {
	now := time.Now()
	rlA, mr := newTestRedisLimiter(t, 1, &now)
	rlB := &redisLimiter{
		rdb:    redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		prefix: rlA.prefix, window: time.Minute, limit: 1, timeout: time.Second,
		now: func() time.Time { return now }, fallback: newLimiter(1, 1),
	}
	t.Cleanup(func() { _ = rlB.rdb.Close() })
	if ok, _ := rlA.allow("k"); !ok {
		t.Fatal("instance A first request should pass")
	}
	if ok, _ := rlB.allow("k"); ok {
		t.Fatal("instance B should see A's usage through the shared store")
	}
}

func TestRedisLimiter_FailsOpenToFallback(t *testing.T) {
	now := time.Now()
	rl, mr := newTestRedisLimiter(t, 1, &now)
	mr.Close() // Redis now unreachable; allow() must delegate to the fallback.
	if ok, _ := rl.allow("k"); !ok {
		t.Fatal("should fail open to the in-memory fallback")
	}
}

func TestUserLimiter_NilClientIsInMemory(t *testing.T) {
	a := userLimiter(nil, "chat", 2, 2, 200*time.Millisecond)
	if _, ok := a.(*limiter); !ok {
		t.Fatalf("want *limiter, got %T", a)
	}
}
