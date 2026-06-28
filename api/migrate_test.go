package main

import (
	"context"
	"testing"
)

func TestMigrateDSN_UnixSocket(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_USER", "app")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "chat")
	t.Setenv("DB_HOST", "/cloudsql/proj:region:inst")
	t.Setenv("DB_PORT", "5432")

	got := migrateDSN()
	want := "postgres://app:secret@/chat?host=%2Fcloudsql%2Fproj%3Aregion%3Ainst"
	if got != want {
		t.Fatalf("migrateDSN()\n got: %s\nwant: %s", got, want)
	}
}

// runMigrations (run in TestMain) must have created the full app schema.
func TestRunMigrations_AppliesSchema(t *testing.T) {
	want := []string{"users", "refresh_tokens", "conversations", "messages", "token_usage"}
	for _, table := range want {
		var exists bool
		err := testPool.QueryRow(context.Background(),
			`select exists (select 1 from information_schema.tables
			 where table_schema = 'public' and table_name = $1)`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("expected table %q to exist after migrations", table)
		}
	}
}
