package main

import (
	"database/sql"
	"embed"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// runMigrations applies all up migrations from the embedded SQL files.
func runMigrations(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	goose.SetBaseFS(embeddedMigrations)
	goose.SetDialect("postgres")
	return goose.Up(db, "migrations")
}

// migrateDSN builds the connection string from DB env vars only.
func migrateDSN() string {
	return dsn(Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		DBHost:      os.Getenv("DB_HOST"),
		DBPort:      os.Getenv("DB_PORT"),
		DBUser:      os.Getenv("DB_USER"),
		DBPassword:  os.Getenv("DB_PASSWORD"),
		DBName:      os.Getenv("DB_NAME"),
	})
}
