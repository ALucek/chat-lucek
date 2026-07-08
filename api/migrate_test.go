package main

import (
	"context"
	"testing"
)

// runMigrations (run in TestMain) must have created the full app schema.
func TestRunMigrations_AppliesSchema(t *testing.T) {
	want := []string{"users", "refresh_tokens", "conversations", "messages", "token_usage", "usage_marks"}
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
