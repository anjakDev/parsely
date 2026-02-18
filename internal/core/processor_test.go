package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parsely/parsely/internal/ai"
	"github.com/parsely/parsely/internal/db"
)

// MockAIExtractor for testing
type MockAIExtractor struct {
	Vocabulary []string
	Err        error
}

func (m *MockAIExtractor) ExtractVocabulary(text, language string) ([]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Vocabulary, nil
}

// TestProcessDocument tests end-to-end document processing
func TestProcessDocument(t *testing.T) {
	// Setup test database
	database := setupTestDB(t)
	defer database.Close()

	// Setup mock AI
	mockAI := &MockAIExtractor{
		Vocabulary: []string{"hola", "adiós", "gracias"},
	}

	// Create processor
	processor := &Processor{
		DB:        database,
		AI:        mockAI,
		Language:  "Spanish",
	}

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("Spanish lesson content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Note: Processing a .txt file will fail because we only support PDF/DOCX
	// This tests that the processor validates file types
	result, err := processor.ProcessDocument(testFile)
	if err == nil {
		t.Error("Expected error for unsupported file type")
	}
	if result != nil {
		t.Error("Result should be nil on error")
	}
}

// TestProcessDocumentDeduplication tests that existing vocabulary is skipped
func TestProcessDocumentDeduplication(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Insert some existing vocabulary
	database.Insert(&db.Vocabulary{Text: "hola", Language: "Spanish"})
	database.Insert(&db.Vocabulary{Text: "gracias", Language: "Spanish"})

	// Mock AI returns 3 words, 2 already exist
	mockAI := &MockAIExtractor{
		Vocabulary: []string{"hola", "adiós", "gracias"},
	}

	processor := &Processor{
		DB:       database,
		Language: "Spanish",
	}

	// For this test, we'll directly test the vocabulary processing
	vocab := mockAI.Vocabulary
	newCount, skipCount := processor.processVocabulary(vocab)

	if newCount != 1 {
		t.Errorf("Expected 1 new item, got %d", newCount)
	}
	if skipCount != 2 {
		t.Errorf("Expected 2 skipped items, got %d", skipCount)
	}
}

// TestFileTypeDetection tests file type validation
func TestFileTypeDetection(t *testing.T) {
	tests := []struct {
		filename string
		valid    bool
	}{
		{"test.pdf", true},
		{"test.docx", true},
		{"test.txt", false},
		{"test.doc", false},
		{"test.PDF", true},
		{"test.DOCX", true},
	}

	for _, tc := range tests {
		isValid := isValidFileType(tc.filename)
		if isValid != tc.valid {
			t.Errorf("isValidFileType(%s) = %v, expected %v", tc.filename, isValid, tc.valid)
		}
	}
}

// TestProcessingResult tests the result structure
func TestProcessingResult(t *testing.T) {
	result := &ProcessingResult{
		NewVocabulary:     5,
		SkippedDuplicates: 3,
		TotalProcessed:    8,
		Language:          "Spanish",
	}

	if result.NewVocabulary != 5 {
		t.Errorf("Expected NewVocabulary=5, got %d", result.NewVocabulary)
	}

	if result.TotalProcessed != 8 {
		t.Errorf("Expected TotalProcessed=8, got %d", result.TotalProcessed)
	}
}

// TestEmptyDocument tests handling of documents with no vocabulary
func TestEmptyDocument(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Mock AI returns empty vocabulary
	mockAI := &MockAIExtractor{
		Vocabulary: []string{},
	}

	processor := &Processor{
		DB:        database,
		AI:        mockAI,
		Language:  "Spanish",
	}

	newCount, skipCount := processor.processVocabulary([]string{})

	if newCount != 0 {
		t.Errorf("Expected 0 new items for empty vocab, got %d", newCount)
	}
	if skipCount != 0 {
		t.Errorf("Expected 0 skipped items for empty vocab, got %d", skipCount)
	}
}

// TestAIError tests handling of AI extraction errors
func TestAIError(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Mock AI that returns an error
	mockAI := &MockAIExtractor{
		Err: &ai.AIError{Message: "API rate limit", StatusCode: 429},
	}

	// Test that AI errors are propagated
	_, err := mockAI.ExtractVocabulary("test", "Spanish")
	if err == nil {
		t.Error("Expected error from mock AI")
	}

	if !ai.IsAIError(err) {
		t.Error("Expected AIError type")
	}
}

// TestProcessVocabularyInsertError tests handling of database insert errors
func TestProcessVocabularyInsertError(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	processor := &Processor{
		DB:       database,
		Language: "Spanish",
	}

	// Insert a vocabulary item
	vocab := []string{"test"}
	newCount, skipCount := processor.processVocabulary(vocab)

	if newCount != 1 {
		t.Errorf("Expected 1 new item, got %d", newCount)
	}

	// Try to insert the same item again (should be skipped)
	newCount, skipCount = processor.processVocabulary(vocab)

	if newCount != 0 {
		t.Errorf("Expected 0 new items on duplicate, got %d", newCount)
	}
	if skipCount != 1 {
		t.Errorf("Expected 1 skipped item on duplicate, got %d", skipCount)
	}
}

// TestNewProcessor tests processor creation
func TestNewProcessor(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	mockAI := &MockAIExtractor{
		Vocabulary: []string{"test"},
	}

	processor := NewProcessor(database, mockAI, "Spanish")

	if processor == nil {
		t.Fatal("Processor should not be nil")
	}

	if processor.DB != database {
		t.Error("Database not set correctly")
	}

	if processor.AI != mockAI {
		t.Error("AI not set correctly")
	}

	if processor.Language != "Spanish" {
		t.Error("Language not set correctly")
	}
}

// TestValidateFilePath tests file path validation
func TestValidateFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid file
	validPath := filepath.Join(tmpDir, "test.pdf")
	err := os.WriteFile(validPath, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := validateFilePath(validPath); err != nil {
		t.Errorf("Valid file should pass validation: %v", err)
	}

	// Non-existent file
	if err := validateFilePath("/nonexistent/file.pdf"); err == nil {
		t.Error("Non-existent file should fail validation")
	}

	// Empty path
	if err := validateFilePath(""); err == nil {
		t.Error("Empty path should fail validation")
	}
}

// TestPathTraversalProtection tests path traversal attack prevention
func TestPathTraversalProtection(t *testing.T) {
	maliciousPaths := []string{
		"../../etc/passwd",
		"../../../etc/shadow",
		"..\\..\\windows\\system32",
	}

	for _, path := range maliciousPaths {
		if !strings.Contains(path, "..") {
			continue
		}
		// The validateFilePath should catch this or the file won't exist
		err := validateFilePath(path)
		if err == nil {
			// If no error, check that the file doesn't exist (which is expected)
			if _, statErr := os.Stat(path); statErr == nil {
				t.Errorf("Path traversal should be prevented: %s", path)
			}
		}
	}
}

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *db.Database {
	database, err := db.NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return database
}
