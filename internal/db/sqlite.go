package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents a SQLite database connection
type Database struct {
	conn *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS vocabulary (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    text TEXT UNIQUE NOT NULL,
    language TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_text ON vocabulary(text);
CREATE INDEX IF NOT EXISTS idx_language ON vocabulary(language);
`

// NewDatabase creates a new database connection and initializes the schema
func NewDatabase(dbPath string) (*Database, error) {
	// For in-memory databases, use shared cache mode for concurrent access
	if dbPath == ":memory:" {
		dbPath = "file::memory:?cache=shared"
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings for better concurrency
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)

	// Enable WAL mode for better concurrent access (skip for in-memory)
	if dbPath != "file::memory:?cache=shared" {
		if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
		}
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create schema
	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &Database{conn: conn}, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Insert adds a new vocabulary item to the database
// Returns the ID of the inserted item or an error if it already exists
func (db *Database) Insert(vocab *Vocabulary) (int, error) {
	query := `INSERT INTO vocabulary (text, language) VALUES (?, ?)`
	result, err := db.conn.Exec(query, vocab.Text, vocab.Language)
	if err != nil {
		return 0, fmt.Errorf("failed to insert vocabulary: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return int(id), nil
}

// Get retrieves a vocabulary item by ID
func (db *Database) Get(id int) (*Vocabulary, error) {
	query := `SELECT id, text, language, created_at FROM vocabulary WHERE id = ?`

	var vocab Vocabulary
	err := db.conn.QueryRow(query, id).Scan(
		&vocab.ID,
		&vocab.Text,
		&vocab.Language,
		&vocab.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("vocabulary with ID %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vocabulary: %w", err)
	}

	return &vocab, nil
}

// List retrieves all vocabulary items ordered by creation date (newest first)
func (db *Database) List() ([]*Vocabulary, error) {
	query := `SELECT id, text, language, created_at FROM vocabulary ORDER BY created_at DESC`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list vocabulary: %w", err)
	}
	defer rows.Close()

	var items []*Vocabulary
	for rows.Next() {
		var vocab Vocabulary
		err := rows.Scan(
			&vocab.ID,
			&vocab.Text,
			&vocab.Language,
			&vocab.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vocabulary: %w", err)
		}
		items = append(items, &vocab)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return items, nil
}

// Delete removes a vocabulary item by ID
func (db *Database) Delete(id int) error {
	query := `DELETE FROM vocabulary WHERE id = ?`
	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete vocabulary: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("vocabulary with ID %d not found", id)
	}

	return nil
}

// ExistsText checks if a vocabulary item with the given text already exists
func (db *Database) ExistsText(text string) (bool, error) {
	query := `SELECT COUNT(*) FROM vocabulary WHERE text = ?`

	var count int
	err := db.conn.QueryRow(query, text).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if text exists: %w", err)
	}

	return count > 0, nil
}

// GetByText retrieves a vocabulary item by its text
func (db *Database) GetByText(text string) (*Vocabulary, error) {
	query := `SELECT id, text, language, created_at FROM vocabulary WHERE text = ?`

	var vocab Vocabulary
	err := db.conn.QueryRow(query, text).Scan(
		&vocab.ID,
		&vocab.Text,
		&vocab.Language,
		&vocab.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("vocabulary with text '%s' not found", text)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vocabulary by text: %w", err)
	}

	return &vocab, nil
}

// ExportToJSON exports all vocabulary items to a JSON file
func (db *Database) ExportToJSON(filePath string) error {
	items, err := db.List()
	if err != nil {
		return fmt.Errorf("failed to list vocabulary for export: %w", err)
	}

	// Create file with secure permissions (0600 - owner read/write only)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(items); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// Count returns the total number of vocabulary items
func (db *Database) Count() (int, error) {
	query := `SELECT COUNT(*) FROM vocabulary`

	var count int
	err := db.conn.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count vocabulary: %w", err)
	}

	return count, nil
}

// SearchByLanguage returns all vocabulary items for a specific language
func (db *Database) SearchByLanguage(language string) ([]*Vocabulary, error) {
	query := `SELECT id, text, language, created_at FROM vocabulary WHERE language = ? ORDER BY created_at DESC`

	rows, err := db.conn.Query(query, language)
	if err != nil {
		return nil, fmt.Errorf("failed to search by language: %w", err)
	}
	defer rows.Close()

	var items []*Vocabulary
	for rows.Next() {
		var vocab Vocabulary
		err := rows.Scan(
			&vocab.ID,
			&vocab.Text,
			&vocab.Language,
			&vocab.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vocabulary: %w", err)
		}
		items = append(items, &vocab)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return items, nil
}
