package parser

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ParsePDF extracts text content from a PDF file
func ParsePDF(filePath string) (string, error) {
	// Validate file size first
	if err := ValidateFileSize(filePath); err != nil {
		return "", err
	}

	// Open the PDF file
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer file.Close()

	// Extract text from all pages
	var textBuilder strings.Builder
	totalPages := reader.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		// Get text content from the page
		text, err := page.GetPlainText(nil)
		if err != nil {
			// Log error but continue with other pages
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	content := textBuilder.String()
	if len(content) == 0 {
		return "", fmt.Errorf("no text content found in PDF")
	}

	return strings.TrimSpace(content), nil
}

// ParsePDFFromReader extracts text from a PDF io.Reader (for uploaded files)
func ParsePDFFromReader(reader io.Reader, size int64) (string, error) {
	// Validate size
	if size > MaxFileSize {
		return "", fmt.Errorf("file too large: %d bytes (max: %d bytes)", size, MaxFileSize)
	}

	// Read all content into memory
	content, err := io.ReadAll(io.LimitReader(reader, MaxFileSize+1))
	if err != nil {
		return "", fmt.Errorf("failed to read PDF content: %w", err)
	}

	if len(content) > MaxFileSize {
		return "", fmt.Errorf("file too large: %d bytes (max: %d bytes)", len(content), MaxFileSize)
	}

	// Open PDF from bytes
	pdfReader, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("failed to parse PDF: %w", err)
	}

	// Extract text from all pages
	var textBuilder strings.Builder
	totalPages := pdfReader.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	result := textBuilder.String()
	if len(result) == 0 {
		return "", fmt.Errorf("no text content found in PDF")
	}

	return strings.TrimSpace(result), nil
}
