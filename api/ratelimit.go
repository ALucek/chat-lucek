package main

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	chatRatePerMin = 20
	chatRateBurst  = 20
)

// entry is one key's limiter plus its last-seen time, tracked for eviction.
type entry struct {
	*rate.Limiter
	lastSeen time.Time
}

// limiter is a per-key token-bucket rate limiter; a sweep evicts idle keys.
type limiter struct {
	mu      sync.Mutex
	entries map[string]*entry
	rate    rate.Limit // tokens per second
	burst   int
	now     func() time.Time
	idleTTL time.Duration
}

// newLimiter allows perMinute requests with burst, and starts the sweep.
func newLimiter(perMinute, burst int) *limiter {
	l := &limiter{
		entries: make(map[string]*entry),
		rate:    rate.Limit(float64(perMinute) / 60.0),
		burst:   burst,
		now:     time.Now,
		idleTTL: 10 * time.Minute,
	}
	go l.sweep()
	return l
}

// allow consumes one token for key, reporting the wait until the next token.
func (l *limiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	e, ok := l.entries[key]
	if !ok {
		e = &entry{Limiter: rate.NewLimiter(l.rate, l.burst)}
		l.entries[key] = e
	}
	e.lastSeen = now

	res := e.ReserveN(now, 1)
	if wait := res.DelayFrom(now); wait > 0 {
		res.CancelAt(now)
		return false, wait
	}
	return true, 0
}

// sweep periodically evicts idle keys for the life of the process.
func (l *limiter) sweep() {
	ticker := time.NewTicker(l.idleTTL)
	for range ticker.C {
		l.evict()
	}
}

// evict drops entries untouched for longer than idleTTL.
func (l *limiter) evict() {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := l.now().Add(-l.idleTTL)
	for k, e := range l.entries {
		if e.lastSeen.Before(cutoff) {
			delete(l.entries, k)
		}
	}
}

// middleware rate-limits by the extractor's key; 429 + Retry-After on deny.
func (l *limiter) middleware(key func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, retryAfter := l.allow(key(r))
			if !ok {
				secs := int(retryAfter.Seconds())
				if secs < 1 {
					secs = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(secs))
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
