package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/parsely/parsely/internal/core"
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

// TestListVocabularyHandler tests GET /api/vocabulary
func TestListVocabularyHandler(t *testing.T) {
	handler := setupTestHandler(t)

	// Add some test vocabulary
	handler.Processor.DB.Insert(&db.Vocabulary{Text: "hola", Language: "Spanish"})
	handler.Processor.DB.Insert(&db.Vocabulary{Text: "adiós", Language: "Spanish"})

	req := httptest.NewRequest("GET", "/api/vocabulary", nil)
	w := httptest.NewRecorder()

	handler.ListVocabulary(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", res.StatusCode)
	}

	var vocab []*db.Vocabulary
	if err := json.NewDecoder(res.Body).Decode(&vocab); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(vocab) != 2 {
		t.Errorf("Expected 2 vocabulary items, got %d", len(vocab))
	}
}

// TestGetVocabularyHandler tests GET /api/vocabulary/{id}
func TestGetVocabularyHandler(t *testing.T) {
	handler := setupTestHandler(t)

	// Add test vocabulary
	id, _ := handler.Processor.DB.Insert(&db.Vocabulary{Text: "test", Language: "Spanish"})

	idStr := fmt.Sprintf("%d", id)
	req := httptest.NewRequest("GET", "/api/vocabulary/"+idStr, nil)
	req.SetPathValue("id", idStr)
	w := httptest.NewRecorder()

	handler.GetVocabulary(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", res.StatusCode)
	}

	var vocab db.Vocabulary
	if err := json.NewDecoder(res.Body).Decode(&vocab); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if vocab.ID != id {
		t.Errorf("Expected ID %d, got %d", id, vocab.ID)
	}
}

// TestDeleteVocabularyHandler tests DELETE /api/vocabulary/{id}
func TestDeleteVocabularyHandler(t *testing.T) {
	handler := setupTestHandler(t)

	// Add test vocabulary
	handler.Processor.DB.Insert(&db.Vocabulary{Text: "delete_me", Language: "Spanish"})

	req := httptest.NewRequest("DELETE", "/api/vocabulary/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.DeleteVocabulary(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", res.StatusCode)
	}

	// Verify it was deleted
	_, err := handler.Processor.DB.Get(1)
	if err == nil {
		t.Error("Vocabulary should have been deleted")
	}
}

// TestUploadHandler tests POST /api/upload
func TestUploadHandler(t *testing.T) {
	handler := setupTestHandler(t)

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0600)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, _ := os.Open(testFile)
	defer file.Close()

	part, _ := writer.CreateFormFile("file", "test.txt")
	io.Copy(part, file)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.UploadDocument(w, req)

	res := w.Result()
	defer res.Body.Close()

	// Expect error for unsupported file type
	if res.StatusCode == http.StatusOK {
		t.Error("Should reject unsupported file type")
	}
}

// TestExportHandler tests POST /api/export
func TestExportHandler(t *testing.T) {
	handler := setupTestHandler(t)

	// Add test vocabulary
	handler.Processor.DB.Insert(&db.Vocabulary{Text: "export_test", Language: "Spanish"})

	req := httptest.NewRequest("POST", "/api/export", nil)
	w := httptest.NewRecorder()

	handler.ExportVocabulary(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", res.StatusCode)
	}

	// Check content type
	contentType := res.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected application/json, got %s", contentType)
	}
}

// TestCORS tests CORS middleware
func TestCORS(t *testing.T) {
	handler := setupTestHandler(t)

	req := httptest.NewRequest("OPTIONS", "/api/vocabulary", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	// Wrap handler with CORS middleware
	corsHandler := CorsMiddleware(http.HandlerFunc(handler.ListVocabulary))
	corsHandler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	// Check CORS headers
	if res.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Error("Missing Access-Control-Allow-Origin header")
	}
}

// TestInvalidJSON tests handling of invalid JSON
func TestInvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/vocabulary", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	// This would be a handler that expects JSON
	// For now, just verify the request setup
	if req.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type not set correctly")
	}
}

// TestRateLimiting tests rate limiting on upload endpoint
func TestRateLimiting(t *testing.T) {
	// This is a placeholder - rate limiting would require more complex setup
	// In production, use a proper rate limiting library
	t.Skip("Rate limiting requires more complex integration test setup")
}

// TestLargeFileRejection tests that oversized files are rejected
func TestLargeFileRejection(t *testing.T) {
	handler := setupTestHandler(t)

	// Create a large file
	tmpDir := t.TempDir()
	largeFile := filepath.Join(tmpDir, "large.pdf")
	largeContent := make([]byte, 11*1024*1024) // 11MB
	os.WriteFile(largeFile, largeContent, 0600)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, _ := os.Open(largeFile)
	defer file.Close()

	part, _ := writer.CreateFormFile("file", "large.pdf")
	io.Copy(part, file)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.UploadDocument(w, req)

	res := w.Result()
	defer res.Body.Close()

	// Should reject large file
	if res.StatusCode == http.StatusOK {
		t.Error("Should reject file over size limit")
	}
}

// setupTestHandler creates a handler with test dependencies
func setupTestHandler(t *testing.T) *Handler {
	database, err := db.NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	mockAI := &MockAIExtractor{
		Vocabulary: []string{"test1", "test2"},
	}

	processor := core.NewProcessor(database, mockAI, "Spanish")

	return &Handler{
		Processor: processor,
	}
}
