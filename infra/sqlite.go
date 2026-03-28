package infra

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

// InitSQLite initializes the local SQLite database and creates the necessary tables.
func InitSQLite(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Create table for supported currency metadata
	query := `
	CREATE TABLE IF NOT EXISTS supported_fx_currencies (
		code TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(query); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create supported_fx_currencies table: %w", err)
	}

	log.Printf("📂 SQLite database initialized successfully at: %s", dbPath)
	return db, nil
}
