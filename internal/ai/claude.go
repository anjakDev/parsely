package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AIExtractor defines the interface for vocabulary extraction
type AIExtractor interface {
	ExtractVocabulary(text, language string) ([]string, error)
}

// ClaudeClient implements AIExtractor using Claude API
type ClaudeClient struct {
	client *anthropic.Client
}

// AIError represents an error from the AI API
type AIError struct {
	Message     string
	StatusCode  int
	RequestID   string
	RawResponse string
}

func (e *AIError) Error() string {
	msg := fmt.Sprintf("AI API error (%d): %s", e.StatusCode, e.Message)
	if e.RequestID != "" {
		msg += fmt.Sprintf("\n  request-id: %s", e.RequestID)
	}
	if e.RawResponse != "" {
		msg += fmt.Sprintf("\n  raw: %s", e.RawResponse)
	}
	return msg
}

// IsAIError checks if an error is an AIError
func IsAIError(err error) bool {
	var aiErr *AIError
	return errors.As(err, &aiErr)
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(apiKey string) (*ClaudeClient, error) {
	if err := validateAPIKey(apiKey); err != nil {
		return nil, err
	}

	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &ClaudeClient{
		client: &client,
	}, nil
}

// ExtractVocabulary uses Claude to extract vocabulary from text
func (c *ClaudeClient) ExtractVocabulary(text, language string) ([]string, error) {
	if strings.TrimSpace(text) == "" {
		return []string{}, nil
	}

	prompt := buildPrompt(text, language)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_5_20250929,
		MaxTokens: 2000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	if err != nil {
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) {
			return nil, &AIError{
				Message:     apiErr.Error(),
				StatusCode:  apiErr.StatusCode,
				RequestID:   apiErr.RequestID,
				RawResponse: apiErr.RawJSON(),
			}
		}
		return nil, &AIError{
			Message:    fmt.Sprintf("failed to call Claude API: %v", err),
			StatusCode: 500,
		}
	}

	if len(message.Content) == 0 {
		return []string{}, nil
	}

	var b strings.Builder
	for _, block := range message.Content {
		if block.Type == "text" {
			b.WriteString(block.AsText().Text)
		}
	}

	vocab, err := parseVocabularyResponse(b.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse vocabulary response: %w", err)
	}

	vocab = sanitizeVocabulary(vocab)
	vocab = deduplicateVocabulary(vocab)

	return vocab, nil
}

// buildPrompt constructs the prompt for Claude
func buildPrompt(text, language string) string {
	if language == "" {
		language = "the target language"
	}

	return fmt.Sprintf(`You are a language learning assistant. Extract all vocabulary words and phrases from the following %s language course notes.

Return ONLY a JSON array of unique vocabulary items, each as a simple string. Include:
- Individual words
- Common phrases
- Expressions
- Greetings

Do NOT include:
- Lesson titles
- Section headers
- English translations (only extract the %s text)
- Duplicate entries

Return format: ["word1", "phrase 2", "word3", ...]

Document content:
%s`, language, language, text)
}

// parseVocabularyResponse extracts a string slice from Claude's JSON response,
// handling optional markdown code block wrappers.
func parseVocabularyResponse(response string) ([]string, error) {
	response = strings.TrimSpace(response)

	// Remove markdown code blocks if present
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var vocab []string
	if err := json.Unmarshal([]byte(response), &vocab); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	return vocab, nil
}

// sanitizeVocabulary cleans up vocabulary items by trimming whitespace and removing empty entries
func sanitizeVocabulary(vocab []string) []string {
	cleaned := make([]string, 0, len(vocab))
	for _, word := range vocab {
		word = strings.TrimSpace(word)
		if word != "" {
			cleaned = append(cleaned, word)
		}
	}
	return cleaned
}

// deduplicateVocabulary removes duplicate entries while preserving order
func deduplicateVocabulary(vocab []string) []string {
	seen := make(map[string]bool, len(vocab))
	unique := make([]string, 0, len(vocab))

	for _, word := range vocab {
		if !seen[word] {
			seen[word] = true
			unique = append(unique, word)
		}
	}

	return unique
}

// validateAPIKey checks if the API key is valid
func validateAPIKey(apiKey string) error {
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	return nil
}
