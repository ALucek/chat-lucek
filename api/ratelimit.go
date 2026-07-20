package main

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
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

// allower reports whether a keyed request may proceed, and the wait if not.
type allower interface {
	allow(key string) (bool, time.Duration)
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

// rateLimit answers 429 + Retry-After once the key exceeds its allowance.
func rateLimit(a allower, key func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, retryAfter := a.allow(key(r))
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

// slidingWindow counts a key's requests in a trailing window via a sorted set.
var slidingWindow = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)
if count < limit then
  redis.call('ZADD', key, now, member)
  redis.call('PEXPIRE', key, window)
  return {1, 0}
end
local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
local retry = window
if oldest[2] then
  retry = (tonumber(oldest[2]) + window) - now
  if retry < 0 then retry = 0 end
end
redis.call('PEXPIRE', key, window)
return {0, retry}
`)

// newRedisClient builds a client from a rediss:// (or redis://) URL.
func newRedisClient(rawURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opt), nil
}

// redisLimiter is a global sliding-window limit with an in-memory fallback.
type redisLimiter struct {
	rdb      *redis.Client
	prefix   string
	window   time.Duration
	limit    int
	timeout  time.Duration
	now      func() time.Time
	fallback *limiter
	seq      uint64
}

// allow consumes one slot for key across all instances sharing the store.
func (r *redisLimiter) allow(key string) (bool, time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	now := r.now()
	member := strconv.FormatInt(now.UnixNano(), 10) + "-" +
		strconv.FormatUint(atomic.AddUint64(&r.seq, 1), 10)
	res, err := slidingWindow.Run(ctx, r.rdb, []string{r.prefix + key},
		now.UnixMilli(), r.window.Milliseconds(), r.limit, member).Int64Slice()
	if err != nil || len(res) != 2 {
		slog.Warn("ratelimit: redis unavailable, using in-memory fallback",
			"limiter", r.prefix, "err", err)
		return r.fallback.allow(key)
	}
	if res[0] == 1 {
		return true, 0
	}
	return false, time.Duration(res[1]) * time.Millisecond
}

// userLimiter returns a global limiter when rdb is set, else an in-memory one.
func userLimiter(rdb *redis.Client, name string, perMin, burst int, timeout time.Duration) allower {
	local := newLimiter(perMin, burst)
	if rdb == nil {
		return local
	}
	return &redisLimiter{
		rdb: rdb, prefix: "rl:" + name + ":", window: time.Minute,
		limit: perMin, timeout: timeout, now: time.Now, fallback: local,
	}
}
