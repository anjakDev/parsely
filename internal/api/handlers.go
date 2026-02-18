package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/parsely/parsely/internal/core"
	"github.com/parsely/parsely/internal/parser"
)

// Handler contains all HTTP handlers.
type Handler struct {
	Processor *core.Processor
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a success response.
type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ListVocabulary handles GET /api/vocabulary.
func (h *Handler) ListVocabulary(w http.ResponseWriter, r *http.Request) {
	vocab, err := h.Processor.GetVocabularyList()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list vocabulary: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, vocab)
}

// GetVocabulary handles GET /api/vocabulary/{id}.
func (h *Handler) GetVocabulary(w http.ResponseWriter, r *http.Request) {
	id, ok := parseVocabularyID(w, r)
	if !ok {
		return
	}

	vocab, err := h.Processor.DB.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Vocabulary not found")
		return
	}

	respondJSON(w, http.StatusOK, vocab)
}

// DeleteVocabulary handles DELETE /api/vocabulary/{id}.
func (h *Handler) DeleteVocabulary(w http.ResponseWriter, r *http.Request) {
	id, ok := parseVocabularyID(w, r)
	if !ok {
		return
	}

	if err := h.Processor.DeleteVocabulary(id); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Vocabulary deleted successfully"})
}

// UploadDocument handles POST /api/upload.
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	if err := parser.ValidateFilename(header.Filename); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid filename: %v", err))
		return
	}

	if header.Size > parser.MaxFileSize {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("File too large (max %d bytes)", parser.MaxFileSize))
		return
	}

	tmpPath, err := parser.CreateTempFile(file, header.Filename)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save file: %v", err))
		return
	}
	defer parser.CleanupTempFile(tmpPath)

	result, err := h.Processor.ProcessDocument(tmpPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process document: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ExportVocabulary handles POST /api/export.
func (h *Handler) ExportVocabulary(w http.ResponseWriter, r *http.Request) {
	vocab, err := h.Processor.GetVocabularyList()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get vocabulary: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=vocabulary_export.json")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(vocab); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to encode JSON: %v", err))
		return
	}
}

// GetStats handles GET /api/stats.
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	count, err := h.Processor.GetVocabularyCount()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get stats: %v", err))
		return
	}

	stats := map[string]any{
		"total_vocabulary": count,
	}

	respondJSON(w, http.StatusOK, stats)
}

// parseVocabularyID extracts and validates the "id" path parameter.
// Returns the parsed ID and true on success, or writes an error response and returns false.
func parseVocabularyID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return 0, false
	}
	return id, true
}

// respondJSON sends a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// respondError sends an error JSON response with the given status code and message.
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ErrorResponse{Error: message})
}

// CorsMiddleware adds CORS headers.
// In production, restrict Access-Control-Allow-Origin to specific origins.
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// RecoverMiddleware recovers from panics and returns a 500 error.
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				respondError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
