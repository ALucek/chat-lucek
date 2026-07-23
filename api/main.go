package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/api/idtoken"
)

const serverIdleTimeout = 120 * time.Second

// WriteTimeout is left unset on purpose cuz streaming SSE response.
func newServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       serverIdleTimeout,
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := runMigrations(migrateDSN()); err != nil {
			slog.Error("migrate", "err", err)
			os.Exit(1)
		}
		slog.Info("migrations applied")
		return
	}

	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	setupLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := NewPool(ctx, cfg)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	var rdb *redis.Client
	if cfg.UpstashRedisURL != "" {
		rdb, err = newRedisClient(cfg.UpstashRedisURL)
		if err != nil {
			slog.Error("redis", "err", err)
			os.Exit(1)
		}
		defer rdb.Close()
	}

	check := func(ctx context.Context) error { return Healthy(ctx, pool) }

	auth := &Auth{pool: pool, secret: []byte(cfg.JWTSecret), verify: selectGoogleVerifier(cfg), exchange: selectGoogleExchanger(cfg), signupOpen: cfg.SignupOpen, mailer: selectMailer(cfg), linkBase: cfg.AllowedOrigin, allowedOrigins: cfg.AllowedOrigins}
	candURL := cfg.AgentCandURL
	if candURL == "" {
		candURL = candURLFor(cfg.AgentURL)
	}
	agent := &agentClient{baseURL: cfg.AgentURL, candURL: candURL, http: newAgentHTTPClient()}
	if strings.HasPrefix(cfg.AgentURL, "https") {
		ts, err := idtoken.NewTokenSource(ctx, cfg.AgentURL)
		if err != nil {
			slog.Error("agent id token", "err", err)
			os.Exit(1)
		}
		agent.token = func(context.Context) (string, error) {
			t, err := ts.Token()
			if err != nil {
				return "", err
			}
			return t.AccessToken, nil
		}
	}
	chat := &Chat{pool: pool, agent: agent, runsBudget: cfg.RunsBudgetDaily, ownerEmail: normalizeEmail(cfg.OwnerEmail), usageSecret: []byte(cfg.UsageHashSecret), mirror: newLangsmithClient(cfg.LangsmithEndpoint, cfg.LangsmithAPIKey), devHost: cfg.DevHost}
	account := &Account{pool: pool}

	mux := newMux(check, auth, chat, account,
		withRateLimitBackend(rdb, time.Duration(cfg.RateLimitTimeoutMS)*time.Millisecond))

	handler := withRequestID(withLogging(withRecover(withSecurityHeaders(withCORS(cfg.AllowedOrigins, withOriginCheck(cfg.AllowedOrigins, withMaxBody(withMaintenance(cfg.Maintenance, mux))))))))
	server := newServer(":"+cfg.Port, handler)

	go func() {
		slog.Info("listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}

// readyHandler reports 200 when the dependency check passes, 503 when it fails.
func readyHandler(check func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := check(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// liveHandler reports 200 while the process serves; no dependency checks.
func liveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

const maxBodyBytes = 1 << 20 // 1 MiB

// withMaxBody caps the request body so a single request can't exhaust memory.
func withMaxBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// withSecurityHeaders sets baseline security headers on every response.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// decodeJSON reads the (size-capped) request body into dst.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request too large"})
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}
		return false
	}
	return true
}

// muxOptions configures optional newMux behavior.
type muxOptions struct {
	rdb       *redis.Client
	rlTimeout time.Duration
}

// muxOption sets a field on muxOptions.
type muxOption func(*muxOptions)

// withRateLimitBackend routes limiters through Redis when rdb is non-nil.
func withRateLimitBackend(rdb *redis.Client, timeout time.Duration) muxOption {
	return func(o *muxOptions) { o.rdb = rdb; o.rlTimeout = timeout }
}

// newMux registers every route.
func newMux(check func(context.Context) error, auth *Auth, chat *Chat, account *Account, opts ...muxOption) *http.ServeMux {
	o := muxOptions{rlTimeout: 200 * time.Millisecond}
	for _, fn := range opts {
		fn(&o)
	}
	protect := func(h http.HandlerFunc) http.Handler { return auth.Middleware(http.HandlerFunc(h)) }

	chatLimiter := userLimiter(o.rdb, "chat", chatRatePerMin, chatRateBurst, o.rlTimeout)
	limitUser := rateLimit(chatLimiter, func(r *http.Request) string {
		uid, _ := userIDFromContext(r.Context())
		return strconv.FormatInt(uid, 10)
	})

	exportLimiter := userLimiter(o.rdb, "export", exportRatePerMin, exportRateBurst, o.rlTimeout)
	limitExport := rateLimit(exportLimiter, func(r *http.Request) string {
		uid, _ := userIDFromContext(r.Context())
		return strconv.FormatInt(uid, 10)
	})

	auth.magicLimiter = userLimiter(o.rdb, "magic", magicRatePerMin, magicRateBurst, o.rlTimeout)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", liveHandler())
	mux.HandleFunc("GET /readyz", readyHandler(check))
	mux.HandleFunc("GET /agentz", chat.Agentz)
	mux.Handle("POST /api/google", http.HandlerFunc(auth.Google))
	mux.Handle("POST /api/magic/request", http.HandlerFunc(auth.MagicRequest))
	mux.Handle("POST /api/magic/verify", http.HandlerFunc(auth.MagicVerify))
	mux.Handle("GET /api/magic/latest", http.HandlerFunc(auth.MagicLatest))
	mux.Handle("POST /api/refresh", http.HandlerFunc(auth.Refresh))
	mux.HandleFunc("POST /api/logout", auth.Logout)
	mux.Handle("GET /api/me", auth.Middleware(http.HandlerFunc(auth.Me)))
	mux.Handle("GET /api/conversations", protect(chat.List))
	mux.Handle("POST /api/conversations", protect(chat.Create))
	mux.Handle("GET /api/conversations/{id}/messages", protect(chat.Messages))
	mux.Handle("PATCH /api/conversations/{id}", protect(chat.Rename))
	mux.Handle("DELETE /api/conversations/{id}", protect(chat.Delete))
	mux.Handle("GET /api/usage", protect(chat.Usage))
	mux.Handle("GET /api/account/export",
		auth.Middleware(limitExport(http.HandlerFunc(account.Export))))
	mux.Handle("DELETE /api/account", protect(account.Delete))
	mux.Handle("POST /api/messages/{id}/feedback", protect(chat.Feedback))
	mux.Handle("DELETE /api/messages/{id}/feedback", protect(chat.ClearFeedback))
	// auth first (puts user in context) → then the user-keyed limiter → handler.
	mux.Handle("POST /api/conversations/{id}/messages",
		auth.Middleware(limitUser(http.HandlerFunc(chat.Send))))
	return mux
}
