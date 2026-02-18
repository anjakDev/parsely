package parser

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nguyenthenguyen/docx"
)

// ParseDOCX extracts text content from a DOCX file
func ParseDOCX(filePath string) (string, error) {
	// Validate file size first
	if err := ValidateFileSize(filePath); err != nil {
		return "", err
	}

	// Read the DOCX file
	doc, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open DOCX: %w", err)
	}
	defer doc.Close()

	// Extract text content
	content := doc.Editable()
	text := content.GetContent()

	if len(strings.TrimSpace(text)) == 0 {
		return "", fmt.Errorf("no text content found in DOCX")
	}

	return strings.TrimSpace(text), nil
}

// ParseDOCXFromReader extracts text from a DOCX by creating a temp file
func ParseDOCXFromReader(reader io.Reader, filename string) (string, error) {
	// Create temporary file
	tmpPath, err := CreateTempFile(reader, filename)
	if err != nil {
		return "", err
	}
	defer CleanupTempFile(tmpPath)

	// Parse the temporary file
	return ParseDOCX(tmpPath)
}

// CreateTempFile creates a temporary file from an io.Reader (for web uploads)
func CreateTempFile(reader io.Reader, filename string) (string, error) {
	// Validate filename
	if err := ValidateFilename(filename); err != nil {
		return "", err
	}

	// Create temp directory if it doesn't exist
	tmpDir := os.TempDir()
	tempFile, err := os.CreateTemp(tmpDir, "parsely-*-"+filename)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Copy content with size limit
	written, err := io.Copy(tempFile, io.LimitReader(reader, MaxFileSize+1))
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if written > MaxFileSize {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("file too large: %d bytes (max: %d bytes)", written, MaxFileSize)
	}

	return tempFile.Name(), nil
}

// CleanupTempFile removes a temporary file
func CleanupTempFile(filePath string) error {
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove temp file: %w", err)
	}
	return nil
}
