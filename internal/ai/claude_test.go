package ai

import (
	"fmt"
	"strings"
	"testing"
)

// MockAIExtractor is a mock implementation for testing
type MockAIExtractor struct {
	ShouldError bool
	Response    []string
}

func (m *MockAIExtractor) ExtractVocabulary(text, language string) ([]string, error) {
	if m.ShouldError {
		return nil, &AIError{Message: "mock error", StatusCode: 500}
	}
	return m.Response, nil
}

// TestExtractVocabulary tests basic vocabulary extraction
func TestExtractVocabulary(t *testing.T) {
	mock := &MockAIExtractor{
		Response: []string{"hola", "buenos días", "gracias"},
	}

	vocab, err := mock.ExtractVocabulary("Some Spanish text", "es")
	if err != nil {
		t.Fatalf("Failed to extract vocabulary: %v", err)
	}

	if len(vocab) != 3 {
		t.Errorf("Expected 3 vocabulary items, got %d", len(vocab))
	}

	expected := map[string]bool{
		"hola":         true,
		"buenos días":  true,
		"gracias":      true,
	}

	for _, word := range vocab {
		if !expected[word] {
			t.Errorf("Unexpected vocabulary item: %s", word)
		}
	}
}

// TestExtractVocabularyError tests error handling
func TestExtractVocabularyError(t *testing.T) {
	mock := &MockAIExtractor{
		ShouldError: true,
	}

	_, err := mock.ExtractVocabulary("Some text", "es")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if !IsAIError(err) {
		t.Error("Expected AIError type")
	}
}

// TestPromptConstruction tests that prompts are well-formed
func TestPromptConstruction(t *testing.T) {
	text := "Spanish lesson content"
	language := "Spanish"

	prompt := buildPrompt(text, language)

	// Check that prompt contains necessary components
	if !strings.Contains(prompt, "vocabulary") {
		t.Error("Prompt should mention 'vocabulary'")
	}

	if !strings.Contains(prompt, language) {
		t.Error("Prompt should mention the target language")
	}

	if !strings.Contains(prompt, text) {
		t.Error("Prompt should contain the document text")
	}

	if !strings.Contains(prompt, "JSON") {
		t.Error("Prompt should mention JSON format")
	}
}

// TestEmptyText tests handling of empty input
func TestEmptyText(t *testing.T) {
	mock := &MockAIExtractor{
		Response: []string{},
	}

	vocab, err := mock.ExtractVocabulary("", "es")
	if err != nil {
		t.Errorf("Should handle empty text: %v", err)
	}

	if len(vocab) != 0 {
		t.Errorf("Expected 0 items for empty text, got %d", len(vocab))
	}
}

// TestParseVocabularyResponse tests parsing JSON responses
func TestParseVocabularyResponse(t *testing.T) {
	tests := []struct {
		name        string
		jsonResp    string
		expected    int
		expectError bool
	}{
		{
			name:        "Valid array",
			jsonResp:    `["word1", "word2", "word3"]`,
			expected:    3,
			expectError: false,
		},
		{
			name:        "Empty array",
			jsonResp:    `[]`,
			expected:    0,
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			jsonResp:    `not json`,
			expected:    0,
			expectError: true,
		},
		{
			name:        "Not an array",
			jsonResp:    `{"key": "value"}`,
			expected:    0,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vocab, err := parseVocabularyResponse(tc.jsonResp)

			if tc.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tc.expectError && len(vocab) != tc.expected {
				t.Errorf("Expected %d items, got %d", tc.expected, len(vocab))
			}
		})
	}
}

// TestDeduplication tests that duplicates are removed
func TestDeduplication(t *testing.T) {
	vocab := []string{"hello", "world", "hello", "goodbye", "world", "hello"}
	deduplicated := deduplicateVocabulary(vocab)

	if len(deduplicated) != 3 {
		t.Errorf("Expected 3 unique items, got %d", len(deduplicated))
	}

	seen := make(map[string]bool)
	for _, word := range deduplicated {
		if seen[word] {
			t.Errorf("Duplicate found after deduplication: %s", word)
		}
		seen[word] = true
	}
}

// TestSanitizeVocabulary tests cleaning vocabulary items
func TestSanitizeVocabulary(t *testing.T) {
	vocab := []string{
		"  hello  ",
		"world",
		"",
		"   ",
		"good morning",
	}

	sanitized := sanitizeVocabulary(vocab)

	// Should remove empty strings and trim whitespace
	if len(sanitized) != 3 {
		t.Errorf("Expected 3 items after sanitization, got %d", len(sanitized))
	}

	for _, word := range sanitized {
		if word != strings.TrimSpace(word) {
			t.Errorf("Word not trimmed: '%s'", word)
		}
		if word == "" {
			t.Error("Empty string should be filtered out")
		}
	}
}

// TestValidateAPIKey tests API key validation
func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		apiKey string
		valid  bool
	}{
		{"sk-ant-1234567890abcdef", true},
		{"", false},
		{"   ", false},
		{"invalid", true}, // We don't validate format, just presence
	}

	for _, tc := range tests {
		err := validateAPIKey(tc.apiKey)
		hasError := err != nil

		if tc.valid && hasError {
			t.Errorf("API key '%s' should be valid but got error: %v", tc.apiKey, err)
		}

		if !tc.valid && !hasError {
			t.Errorf("API key '%s' should be invalid but got no error", tc.apiKey)
		}
	}
}

// TestClaudeClientCreation tests creating a Claude client
func TestClaudeClientCreation(t *testing.T) {
	// Test with empty API key
	_, err := NewClaudeClient("")
	if err == nil {
		t.Error("Should fail with empty API key")
	}

	// Test with valid API key format
	client, err := NewClaudeClient("test-key-123")
	if err != nil {
		t.Errorf("Should succeed with non-empty API key: %v", err)
	}

	if client == nil {
		t.Error("Client should not be nil")
	}
}

// TestAIErrorInterface tests the AIError type
func TestAIErrorInterface(t *testing.T) {
	err := &AIError{
		Message:    "test error",
		StatusCode: 429,
	}

	if err.Error() != "AI API error (429): test error" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}

	if !IsAIError(err) {
		t.Error("IsAIError should return true for AIError")
	}

	regularErr := fmt.Errorf("regular error")
	if IsAIError(regularErr) {
		t.Error("IsAIError should return false for non-AIError")
	}
}
