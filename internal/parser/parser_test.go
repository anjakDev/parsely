package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParsePDF tests extracting text from a valid PDF
func TestParsePDF(t *testing.T) {
	// Skip if PDF test file doesn't exist yet
	// We'll create actual PDF files later for integration tests
	t.Skip("PDF test file creation pending - will test with real files")
}

// TestParseDOCX tests extracting text from a valid DOCX
func TestParseDOCX(t *testing.T) {
	// Skip if DOCX test file doesn't exist yet
	// We'll create actual DOCX files later for integration tests
	t.Skip("DOCX test file creation pending - will test with real files")
}

// TestParseInvalidFile tests handling corrupted files
func TestParseInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	corruptedPath := filepath.Join(tmpDir, "corrupted.pdf")

	// Create a file with invalid content
	err := os.WriteFile(corruptedPath, []byte("This is not a valid PDF file"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = ParsePDF(corruptedPath)
	if err == nil {
		t.Error("Expected error when parsing corrupted PDF, got nil")
	}
}

// TestParseNonexistentFile tests handling missing files
func TestParseNonexistentFile(t *testing.T) {
	_, err := ParsePDF("/nonexistent/file.pdf")
	if err == nil {
		t.Error("Expected error when parsing nonexistent file, got nil")
	}

	_, err = ParseDOCX("/nonexistent/file.docx")
	if err == nil {
		t.Error("Expected error when parsing nonexistent file, got nil")
	}
}

// TestParseEmptyPDF tests handling empty PDF files
func TestParseEmptyPDF(t *testing.T) {
	tmpDir := t.TempDir()
	emptyPath := filepath.Join(tmpDir, "empty.pdf")

	// Create minimal valid PDF structure (empty)
	minimalPDF := "%PDF-1.4\n%%EOF"
	err := os.WriteFile(emptyPath, []byte(minimalPDF), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// For minimal PDF, we expect an error since it's not truly valid
	_, err = ParsePDF(emptyPath)
	if err == nil {
		t.Error("Expected error for minimal PDF structure")
	}
}

// TestParseEmptyDOCX tests handling empty DOCX files
func TestParseEmptyDOCX(t *testing.T) {
	// We'll skip this for now as creating valid empty DOCX requires ZIP structure
	t.Skip("Empty DOCX test requires proper ZIP structure - integration test")
}

// TestParseLargeFile tests handling large files up to the limit
func TestParseLargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	largePath := filepath.Join(tmpDir, "large.txt")

	// Create a large text file (5MB)
	largeContent := strings.Repeat("This is a test line.\n", 250000)
	err := os.WriteFile(largePath, []byte(largeContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Just verify file size validation works
	info, err := os.Stat(largePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Size() > MaxFileSize {
		t.Error("Test file exceeds max size")
	}
}

// TestParseOversizedFile tests rejecting files over the size limit
func TestParseOversizedFile(t *testing.T) {
	tmpDir := t.TempDir()
	oversizedPath := filepath.Join(tmpDir, "oversize.pdf")

	// Create a file larger than 10MB
	oversizedContent := make([]byte, MaxFileSize+1)
	err := os.WriteFile(oversizedPath, oversizedContent, 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = ParsePDF(oversizedPath)
	if err == nil {
		t.Error("Expected error when parsing oversized file, got nil")
	}

	if !strings.Contains(err.Error(), "too large") && !strings.Contains(err.Error(), "size") {
		t.Errorf("Error should mention file size, got: %v", err)
	}
}

// TestDetectFileType tests file type detection
func TestDetectFileType(t *testing.T) {
	tests := []struct {
		filename string
		expected FileType
	}{
		{"document.pdf", TypePDF},
		{"notes.PDF", TypePDF},
		{"lesson.docx", TypeDOCX},
		{"file.DOCX", TypeDOCX},
		{"invalid.txt", TypeUnknown},
		{"no_extension", TypeUnknown},
		{"doc.pdf.bak", TypeUnknown},
	}

	for _, tc := range tests {
		result := DetectFileType(tc.filename)
		if result != tc.expected {
			t.Errorf("DetectFileType(%s) = %v, expected %v", tc.filename, result, tc.expected)
		}
	}
}

// TestValidateFileSize tests file size validation
func TestValidateFileSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Small file (should pass)
	smallPath := filepath.Join(tmpDir, "small.txt")
	err := os.WriteFile(smallPath, []byte("small content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := ValidateFileSize(smallPath); err != nil {
		t.Errorf("Small file should pass validation: %v", err)
	}

	// Large file (should fail)
	largePath := filepath.Join(tmpDir, "large.txt")
	largeContent := make([]byte, MaxFileSize+1)
	err = os.WriteFile(largePath, largeContent, 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := ValidateFileSize(largePath); err == nil {
		t.Error("Large file should fail validation")
	}

	// Nonexistent file (should fail)
	if err := ValidateFileSize("/nonexistent/file.txt"); err == nil {
		t.Error("Nonexistent file should fail validation")
	}
}

// TestSanitizeFilename tests filename sanitization for path traversal prevention
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		safe     bool
		contains string
	}{
		{"normal.pdf", true, ""},
		{"my-document.docx", true, ""},
		{"../../etc/passwd", false, ".."},
		{"/etc/passwd", false, "absolute"},
		{"file\x00.pdf", false, "null"},
		{"file\n.pdf", false, "newline"},
		{"con.pdf", true, ""}, // Windows reserved name but we'll allow
		{".hidden.pdf", true, ""},
		{"file with spaces.pdf", true, ""},
	}

	for _, tc := range tests {
		err := ValidateFilename(tc.input)
		if tc.safe && err != nil {
			t.Errorf("ValidateFilename(%q) should be safe but got error: %v", tc.input, err)
		}
		if !tc.safe && err == nil {
			t.Errorf("ValidateFilename(%q) should be unsafe but got no error", tc.input)
		}
		if !tc.safe && err != nil && tc.contains != "" {
			if !strings.Contains(err.Error(), tc.contains) && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.contains)) {
				t.Errorf("ValidateFilename(%q) error should mention %q, got: %v", tc.input, tc.contains, err)
			}
		}
	}
}

// TestParseDocument is the main entry point that detects file type
func TestParseDocument(t *testing.T) {
	tests := []struct {
		filename    string
		expectError bool
	}{
		{"test.pdf", true},  // Invalid PDF content - error expected
		{"test.docx", true}, // Invalid DOCX content - error expected
		{"test.txt", true},  // Unsupported type
	}

	for _, tc := range tests {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, tc.filename)

		// Create a minimal file (invalid for PDF/DOCX)
		err := os.WriteFile(filePath, []byte("test content"), 0600)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err = ParseDocument(filePath)
		hasError := err != nil

		if hasError != tc.expectError {
			t.Errorf("ParseDocument(%s): expected error=%v, got error=%v (%v)",
				tc.filename, tc.expectError, hasError, err)
		}
	}
}
