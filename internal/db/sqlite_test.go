package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestInitializeDatabase tests database initialization
func TestInitializeDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Check if database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify table exists by attempting to insert
	vocab := &Vocabulary{
		Text:     "test",
		Language: "en",
	}
	_, err = db.Insert(vocab)
	if err != nil {
		t.Errorf("Table creation failed: %v", err)
	}
}

// TestInsertVocabulary tests inserting new vocabulary
func TestInsertVocabulary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vocab := &Vocabulary{
		Text:     "hello",
		Language: "en",
	}

	id, err := db.Insert(vocab)
	if err != nil {
		t.Fatalf("Failed to insert vocabulary: %v", err)
	}

	if id <= 0 {
		t.Error("Expected positive ID after insert")
	}

	// Verify it was inserted
	retrieved, err := db.Get(id)
	if err != nil {
		t.Fatalf("Failed to retrieve inserted vocabulary: %v", err)
	}

	if retrieved.Text != vocab.Text {
		t.Errorf("Expected text '%s', got '%s'", vocab.Text, retrieved.Text)
	}
	if retrieved.Language != vocab.Language {
		t.Errorf("Expected language '%s', got '%s'", vocab.Language, retrieved.Language)
	}
}

// TestInsertDuplicate tests that duplicate text is rejected
func TestInsertDuplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vocab := &Vocabulary{
		Text:     "unique",
		Language: "en",
	}

	// First insert should succeed
	_, err := db.Insert(vocab)
	if err != nil {
		t.Fatalf("First insert failed: %v", err)
	}

	// Second insert with same text should fail
	_, err = db.Insert(vocab)
	if err == nil {
		t.Error("Expected error when inserting duplicate, got nil")
	}
}

// TestGetVocabulary tests retrieving a single vocabulary item
func TestGetVocabulary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vocab := &Vocabulary{
		Text:     "world",
		Language: "es",
	}

	id, err := db.Insert(vocab)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	retrieved, err := db.Get(id)
	if err != nil {
		t.Fatalf("Failed to get vocabulary: %v", err)
	}

	if retrieved.ID != id {
		t.Errorf("Expected ID %d, got %d", id, retrieved.ID)
	}
	if retrieved.Text != vocab.Text {
		t.Errorf("Expected text '%s', got '%s'", vocab.Text, retrieved.Text)
	}
}

// TestGetNonexistent tests retrieving a non-existent item
func TestGetNonexistent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Get(99999)
	if err == nil {
		t.Error("Expected error when getting non-existent item, got nil")
	}
}

// TestListVocabulary tests listing all vocabulary items
func TestListVocabulary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert multiple items
	items := []string{"one", "two", "three"}
	for _, text := range items {
		vocab := &Vocabulary{
			Text:     text,
			Language: "en",
		}
		_, err := db.Insert(vocab)
		if err != nil {
			t.Fatalf("Failed to insert '%s': %v", text, err)
		}
	}

	// List all
	all, err := db.List()
	if err != nil {
		t.Fatalf("Failed to list vocabulary: %v", err)
	}

	if len(all) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(all))
	}
}

// TestDeleteVocabulary tests deleting a vocabulary item
func TestDeleteVocabulary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vocab := &Vocabulary{
		Text:     "delete_me",
		Language: "en",
	}

	id, err := db.Insert(vocab)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Delete it
	err = db.Delete(id)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify it's gone
	_, err = db.Get(id)
	if err == nil {
		t.Error("Expected error when getting deleted item, got nil")
	}
}

// TestExistsText tests checking if text already exists
func TestExistsText(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vocab := &Vocabulary{
		Text:     "exists_test",
		Language: "en",
	}

	// Should not exist initially
	exists, err := db.ExistsText(vocab.Text)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Text should not exist before insert")
	}

	// Insert it
	_, err = db.Insert(vocab)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Should exist now
	exists, err = db.ExistsText(vocab.Text)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Text should exist after insert")
	}
}

// TestSQLInjection tests that parameterized queries prevent SQL injection
func TestSQLInjection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Attempt SQL injection in text field
	maliciousText := "'; DROP TABLE vocabulary; --"
	vocab := &Vocabulary{
		Text:     maliciousText,
		Language: "en",
	}

	id, err := db.Insert(vocab)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Verify the malicious text was stored as-is (not executed)
	retrieved, err := db.Get(id)
	if err != nil {
		t.Fatalf("Failed to retrieve: %v", err)
	}

	if retrieved.Text != maliciousText {
		t.Errorf("Text was modified, possible injection: got '%s'", retrieved.Text)
	}

	// Verify table still exists by listing
	_, err = db.List()
	if err != nil {
		t.Error("Table was dropped, SQL injection vulnerability exists!")
	}
}

// TestExportToJSON tests exporting database to JSON
func TestExportToJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert test data
	items := []string{"export1", "export2", "export3"}
	for _, text := range items {
		vocab := &Vocabulary{
			Text:     text,
			Language: "en",
		}
		_, err := db.Insert(vocab)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// Export to JSON
	tmpDir := t.TempDir()
	exportPath := filepath.Join(tmpDir, "export.json")

	err := db.ExportToJSON(exportPath)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Error("Export file was not created")
	}

	// Read and verify content (basic check)
	content, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Export file is empty")
	}
}

// TestConcurrentInserts tests concurrent inserts for race conditions
func TestConcurrentInserts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	done := make(chan error, 10)
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			vocab := &Vocabulary{
				Text:     fmt.Sprintf("concurrent_%d", n),
				Language: "en",
			}
			_, err := db.Insert(vocab)
			done <- err
		}(i)
	}

	// Wait for all goroutines and check errors
	var errors []error
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Errorf("Concurrent inserts failed with %d errors: %v", len(errors), errors[0])
	}

	// Verify all were inserted
	all, err := db.List()
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}

	if len(all) != numGoroutines {
		t.Errorf("Expected %d items, got %d", numGoroutines, len(all))
	}
}

// TestCreatedAtTimestamp tests that created_at is set correctly
func TestCreatedAtTimestamp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	before := time.Now().UTC()
	vocab := &Vocabulary{
		Text:     "timestamp_test",
		Language: "en",
	}

	id, err := db.Insert(vocab)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // Small delay to ensure timestamp difference
	after := time.Now().UTC()

	retrieved, err := db.Get(id)
	if err != nil {
		t.Fatalf("Failed to retrieve: %v", err)
	}

	// CreatedAt should be between before and after (with 1 second tolerance for SQLite)
	createdUTC := retrieved.CreatedAt.UTC()
	if createdUTC.Before(before.Add(-1*time.Second)) || createdUTC.After(after.Add(1*time.Second)) {
		t.Errorf("CreatedAt timestamp %v is outside expected range [%v, %v]",
			createdUTC, before, after)
	}
}

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *Database {
	db, err := NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}
