package main

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	authRatePerMin = 10
	authRateBurst  = 10
	chatRatePerMin = 20
	chatRateBurst  = 20
)

// bucket is one key's token bucket.
type bucket struct {
	tokens float64
	last   time.Time
}

// limiter is a per-key token-bucket rate limiter. The map is guarded by mu; a
// background sweep evicts idle keys so it can't grow unbounded.
type limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64
	now     func() time.Time
	idleTTL time.Duration
}

// newLimiter builds a limiter allowing perMinute requests with the given burst,
// and starts the eviction sweep.
func newLimiter(perMinute, burst int) *limiter {
	l := &limiter{
		buckets: make(map[string]*bucket),
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
		now:     time.Now,
		idleTTL: 10 * time.Minute,
	}
	go l.sweep()
	return l
}

// allow consumes one token for key.
func (l *limiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// sweep periodically evicts idle keys for the life of the process.
func (l *limiter) sweep() {
	ticker := time.NewTicker(l.idleTTL)
	for range ticker.C {
		l.evict()
	}
}

// evict drops buckets untouched for longer than idleTTL.
func (l *limiter) evict() {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := l.now().Add(-l.idleTTL)
	for k, b := range l.buckets {
		if b.last.Before(cutoff) {
			delete(l.buckets, k)
		}
	}
}

// middleware rate-limits by the key the extractor returns; 429 + Retry-After on deny.
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

// clientIP returns the rate-limit key: the real client IP. Behind a trusted
// proxy that is the second-from-right X-Forwarded-For entry; leading entries
// are client-spoofable and ignored.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[len(parts)-2])
			}
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
