package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/parsely/parsely/internal/ai"
	"github.com/parsely/parsely/internal/db"
	"github.com/parsely/parsely/internal/parser"
)

// Processor orchestrates document processing
type Processor struct {
	DB       *db.Database
	AI       ai.AIExtractor
	Language string
}

// ProcessingResult contains the results of processing a document
type ProcessingResult struct {
	NewVocabulary     int
	SkippedDuplicates int
	TotalProcessed    int
	Language          string
	FilePath          string
}

// NewProcessor creates a new Processor instance
func NewProcessor(database *db.Database, aiClient ai.AIExtractor, language string) *Processor {
	return &Processor{
		DB:       database,
		AI:       aiClient,
		Language: language,
	}
}

// ProcessDocument processes a document file and extracts vocabulary
func (p *Processor) ProcessDocument(filePath string) (*ProcessingResult, error) {
	if err := validateFilePath(filePath); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	if !isValidFileType(filePath) {
		return nil, fmt.Errorf("unsupported file type: %s (only .pdf and .docx are supported)", filepath.Ext(filePath))
	}

	text, err := parser.ParseDocument(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}

	vocabulary, err := p.AI.ExtractVocabulary(text, p.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to extract vocabulary: %w", err)
	}

	newCount, skipCount := p.processVocabulary(vocabulary)

	return &ProcessingResult{
		NewVocabulary:     newCount,
		SkippedDuplicates: skipCount,
		TotalProcessed:    newCount + skipCount,
		Language:          p.Language,
		FilePath:          filePath,
	}, nil
}

// processVocabulary inserts new vocabulary items and counts duplicates
func (p *Processor) processVocabulary(vocabulary []string) (newCount, skipCount int) {
	for _, word := range vocabulary {
		exists, err := p.DB.ExistsText(word)
		if err != nil {
			continue
		}
		if exists {
			skipCount++
			continue
		}

		_, err = p.DB.Insert(&db.Vocabulary{
			Text:     word,
			Language: p.Language,
		})
		if err != nil {
			// Insert failure (e.g., race condition) is treated as a duplicate
			skipCount++
			continue
		}

		newCount++
	}

	return newCount, skipCount
}

// validateFilePath checks if a file path is valid, exists, and is a regular file
func validateFilePath(filePath string) error {
	if strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file does not exist: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	return nil
}

// isValidFileType checks if the file has a supported extension
func isValidFileType(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".pdf" || ext == ".docx"
}

// GetVocabularyList retrieves all vocabulary from the database
func (p *Processor) GetVocabularyList() ([]*db.Vocabulary, error) {
	return p.DB.List()
}

// GetVocabularyByLanguage retrieves vocabulary for a specific language
func (p *Processor) GetVocabularyByLanguage(language string) ([]*db.Vocabulary, error) {
	return p.DB.SearchByLanguage(language)
}

// ExportVocabulary exports all vocabulary to a JSON file
func (p *Processor) ExportVocabulary(filePath string) error {
	return p.DB.ExportToJSON(filePath)
}

// GetVocabularyCount returns the total number of vocabulary items
func (p *Processor) GetVocabularyCount() (int, error) {
	return p.DB.Count()
}

// DeleteVocabulary removes a vocabulary item by ID
func (p *Processor) DeleteVocabulary(id int) error {
	return p.DB.Delete(id)
}
