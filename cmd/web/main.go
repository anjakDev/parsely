package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/parsely/parsely/internal/ai"
	"github.com/parsely/parsely/internal/api"
	"github.com/parsely/parsely/internal/core"
	"github.com/parsely/parsely/internal/db"
)

func main() {
	// Load environment variables
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: ANTHROPIC_API_KEY environment variable not set")
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "parsely.db"
	}

	language := os.Getenv("LANGUAGE")
	if language == "" {
		language = "auto-detect"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database
	database, err := db.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer database.Close()

	// Initialize AI client
	aiClient, err := ai.NewClaudeClient(apiKey)
	if err != nil {
		log.Fatalf("Error initializing AI client: %v", err)
	}

	// Create processor
	processor := core.NewProcessor(database, aiClient, language)

	// Create API handler
	handler := &api.Handler{
		Processor: processor,
	}

	// Setup router
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/vocabulary", handler.ListVocabulary)
	mux.HandleFunc("GET /api/vocabulary/{id}", handler.GetVocabulary)
	mux.HandleFunc("DELETE /api/vocabulary/{id}", handler.DeleteVocabulary)
	mux.HandleFunc("POST /api/upload", handler.UploadDocument)
	mux.HandleFunc("POST /api/export", handler.ExportVocabulary)
	mux.HandleFunc("GET /api/stats", handler.GetStats)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply middleware
	var handlerWithMiddleware http.Handler = mux
	handlerWithMiddleware = api.CorsMiddleware(handlerWithMiddleware)
	handlerWithMiddleware = api.LoggingMiddleware(handlerWithMiddleware)
	handlerWithMiddleware = api.RecoverMiddleware(handlerWithMiddleware)

	// Start server
	addr := ":" + port
	fmt.Printf("Starting Parsely web server on http://localhost%s\n", addr)
	fmt.Printf("Database: %s\n", dbPath)
	fmt.Printf("Language: %s\n", language)
	fmt.Println("\nAPI Endpoints:")
	fmt.Println("  GET    /api/vocabulary      - List all vocabulary")
	fmt.Println("  GET    /api/vocabulary/{id} - Get vocabulary by ID")
	fmt.Println("  DELETE /api/vocabulary/{id} - Delete vocabulary by ID")
	fmt.Println("  POST   /api/upload          - Upload and process document")
	fmt.Println("  POST   /api/export          - Export vocabulary to JSON")
	fmt.Println("  GET    /api/stats           - Get vocabulary statistics")
	fmt.Println("  GET    /health              - Health check")

	if err := http.ListenAndServe(addr, handlerWithMiddleware); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
