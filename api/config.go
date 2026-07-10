package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds everything the app needs to run, read once from the environment.
type Config struct {
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	Port               string
	JWTSecret          string
	UsageHashSecret    string
	ResendAPIKey       string
	MagicLinkFrom      string
	AgentURL           string
	AllowedOrigin      string
	DatabaseURL        string
	LogLevel           string
	RunsBudgetDaily    int
	GoogleClientID     string
	GoogleClientSecret string
	OwnerEmail         string
	GoogleAuthFake     bool
	SignupOpen         bool
	Maintenance        bool
}

// LoadConfig reads the required settings from the environment.
func LoadConfig() (Config, error) {
	cfg := Config{
		DBHost:             os.Getenv("DB_HOST"),
		DBPort:             os.Getenv("DB_PORT"),
		DBUser:             os.Getenv("DB_USER"),
		DBPassword:         os.Getenv("DB_PASSWORD"),
		DBName:             os.Getenv("DB_NAME"),
		Port:               os.Getenv("PORT"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		UsageHashSecret:    os.Getenv("USAGE_HASH_SECRET"),
		ResendAPIKey:       os.Getenv("RESEND_API_KEY"),
		MagicLinkFrom:      os.Getenv("MAGIC_LINK_FROM"),
		AgentURL:           getenvDefault("AGENT_URL", "http://localhost:8081"),
		AllowedOrigin:      getenvDefault("ALLOWED_ORIGIN", "http://localhost:3000"),
		LogLevel:           getenvDefault("LOG_LEVEL", "info"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		RunsBudgetDaily:    getenvInt("RUNS_BUDGET_DAILY", 20),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		OwnerEmail:         os.Getenv("OWNER_EMAIL"),
		GoogleAuthFake:     os.Getenv("GOOGLE_AUTH_FAKE") == "1",
		SignupOpen:         os.Getenv("SIGNUP_OPEN") == "true",
		Maintenance:        os.Getenv("MAINTENANCE_MODE") == "1",
	}

	required := []struct{ name, value string }{
		{"DB_HOST", cfg.DBHost},
		{"DB_PORT", cfg.DBPort},
		{"DB_USER", cfg.DBUser},
		{"DB_PASSWORD", cfg.DBPassword},
		{"DB_NAME", cfg.DBName},
		{"PORT", cfg.Port},
		{"JWT_SECRET", cfg.JWTSecret},
		{"GOOGLE_CLIENT_ID", cfg.GoogleClientID},
		{"USAGE_HASH_SECRET", cfg.UsageHashSecret},
	}
	for _, r := range required {
		if r.value == "" {
			return Config{}, fmt.Errorf("missing required env var: %s", r.name)
		}
	}

	// The real Google exchanger needs the client secret; fake auth doesn't.
	if !cfg.GoogleAuthFake && cfg.GoogleClientSecret == "" {
		return Config{}, fmt.Errorf("missing required env var: GOOGLE_CLIENT_SECRET")
	}

	// Fake auth is a total bypass; never allow it under TLS (production).
	if cfg.GoogleAuthFake && strings.HasPrefix(cfg.AllowedOrigin, "https://") {
		return Config{}, fmt.Errorf("GOOGLE_AUTH_FAKE must not be set when ALLOWED_ORIGIN is https")
	}

	// A too-short JWT_SECRET makes session tokens forgeable.
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters, got %d", len(cfg.JWTSecret))
	}

	// The usage-ledger HMAC key; short keys weaken the pseudonym.
	if len(cfg.UsageHashSecret) < 32 {
		return Config{}, fmt.Errorf("USAGE_HASH_SECRET must be at least 32 characters, got %d", len(cfg.UsageHashSecret))
	}

	// Magic-link email needs a real mailer in production.
	if cfg.ResendAPIKey == "" && strings.HasPrefix(cfg.AllowedOrigin, "https://") {
		return Config{}, fmt.Errorf("RESEND_API_KEY must be set when ALLOWED_ORIGIN is https")
	}
	// The real mailer needs a verified sender address.
	if cfg.ResendAPIKey != "" && cfg.MagicLinkFrom == "" {
		return Config{}, fmt.Errorf("MAGIC_LINK_FROM must be set when RESEND_API_KEY is set")
	}

	return cfg, nil
}

// getenvDefault returns the env var if set, otherwise def.
func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getenvInt returns the env var parsed as a positive int, otherwise def.
func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
