package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileType represents the type of document file
type FileType int

const (
	TypeUnknown FileType = iota
	TypePDF
	TypeDOCX
)

// MaxFileSize is the maximum allowed file size (10MB)
const MaxFileSize = 10 * 1024 * 1024

// DetectFileType determines the file type based on extension
func DetectFileType(filename string) FileType {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return TypePDF
	case ".docx":
		return TypeDOCX
	default:
		return TypeUnknown
	}
}

// ValidateFileSize checks if a file is within the size limit
func ValidateFileSize(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d bytes)", info.Size(), MaxFileSize)
	}

	return nil
}

// ValidateFilename checks for path traversal and other malicious patterns
func ValidateFilename(filename string) error {
	// Check for path traversal
	if strings.Contains(filename, "..") {
		return fmt.Errorf("filename contains path traversal: ..")
	}

	// Check for absolute paths
	if strings.HasPrefix(filename, "/") || strings.HasPrefix(filename, "\\") {
		return fmt.Errorf("filename cannot be an absolute path")
	}

	// Check for null bytes
	if strings.ContainsRune(filename, '\x00') {
		return fmt.Errorf("filename contains null byte")
	}

	// Check for newlines
	if strings.ContainsRune(filename, '\n') || strings.ContainsRune(filename, '\r') {
		return fmt.Errorf("filename contains newline character")
	}

	return nil
}

// ParseDocument is the main entry point that detects file type and parses accordingly
func ParseDocument(filePath string) (string, error) {
	// Validate file exists
	if _, err := os.Stat(filePath); err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	// Validate file size
	if err := ValidateFileSize(filePath); err != nil {
		return "", err
	}

	// Detect file type
	fileType := DetectFileType(filePath)

	switch fileType {
	case TypePDF:
		return ParsePDF(filePath)
	case TypeDOCX:
		return ParseDOCX(filePath)
	default:
		return "", fmt.Errorf("unsupported file type: %s", filepath.Ext(filePath))
	}
}
