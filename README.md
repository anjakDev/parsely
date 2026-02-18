# Parsely - Language Learning Vocabulary Extractor

Parsely is a tool that uses AI to extract vocabulary from language learning course notes (PDF/DOCX files) and stores them in a searchable database. It features both a command-line interface (TUI) and a web interface.

## Features

- **AI-Powered Extraction**: Uses Claude AI to intelligently extract vocabulary and phrases
- **Document Support**: Parses PDF and DOCX files
- **Deduplication**: Automatically skips vocabulary that's already in the database
- **Dual Interface**: Choose between CLI (Terminal UI) or Web interface
- **Export**: Export vocabulary to JSON for use in other applications
- **Security**: Built with security best practices (SQL injection prevention, file validation, etc.)

## Requirements

- Go 1.23 or later
- Claude API key (get one from [Anthropic](https://console.anthropic.com/))
- Optional: Bun or Node.js for the web frontend (if you want to develop it)

## Installation

### Clone the repository

```bash
git clone https://github.com/parsely/parsely.git
cd parsely
```

### Install dependencies

```bash
go mod download
```

### Build the binaries

```bash
# Build CLI version
go build -o parsely-cli ./cmd/cli

# Build web version
go build -o parsely-web ./cmd/web
```

## Configuration

Parsely uses environment variables for configuration:

```bash
# Required
export ANTHROPIC_API_KEY="your-api-key-here"

# Optional (with defaults)
export DATABASE_PATH="parsely.db"        # Default: parsely.db
export LANGUAGE="Spanish"                # Default: auto-detect
export PORT="8080"                       # Default: 8080 (web only)
```

## Usage

### CLI Version

Run the interactive terminal UI:

```bash
./parsely-cli
```

Features:
- Parse new documents (PDF/DOCX)
- View all vocabulary
- Export to JSON
- Navigate with arrow keys or vim keys (j/k)

### Web Version

Start the web server:

```bash
./parsely-web
```

The API will be available at `http://localhost:8080`

#### API Endpoints

```
GET    /api/vocabulary       - List all vocabulary
GET    /api/vocabulary/{id}  - Get specific vocabulary item
DELETE /api/vocabulary/{id}  - Delete vocabulary item
POST   /api/upload           - Upload and process document
POST   /api/export           - Export vocabulary to JSON
GET    /api/stats            - Get vocabulary statistics
GET    /health               - Health check
```

#### Upload Document Example

```bash
curl -X POST -F "file=@/path/to/document.pdf" http://localhost:8080/api/upload
```

## Running Tests

Run all tests with coverage:

```bash
go test ./... -cover
```

Run tests for a specific package:

```bash
go test ./internal/db -v
go test ./internal/parser -v
go test ./internal/ai -v
go test ./internal/core -v
go test ./internal/api -v
```

## Project Structure

```
parsely/
├── cmd/
│   ├── cli/          # CLI application entry point
│   └── web/          # Web server entry point
├── internal/
│   ├── ai/           # Claude AI integration
│   ├── parser/       # PDF/DOCX parsers
│   ├── db/           # SQLite database layer
│   ├── core/         # Core business logic
│   └── api/          # HTTP API handlers
├── testdata/         # Test fixtures
├── go.mod
├── go.sum
├── README.md
└── CLAUDE.md         # Development guidelines
```

## Security Features

- **SQL Injection Prevention**: All database queries use parameterized statements
- **Path Traversal Protection**: File paths are validated to prevent directory traversal
- **File Size Limits**: Maximum 10MB per document
- **File Type Validation**: Only PDF and DOCX files accepted
- **Input Sanitization**: All user input is validated and sanitized
- **Secure Permissions**: Database and temp files created with restrictive permissions

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests first (TDD approach)
4. Implement your feature
5. Ensure all tests pass
6. Submit a pull request

See CLAUDE.md for detailed development guidelines.

## Troubleshooting

### "ANTHROPIC_API_KEY not set"

Make sure you've exported your API key:
```bash
export ANTHROPIC_API_KEY="your-key"
```

### Database Permission Errors

Ensure the database file has proper permissions:
```bash
chmod 600 parsely.db
```

### PDF Parsing Errors

Some PDFs may not contain extractable text. Try:
1. Ensuring the PDF has selectable text (not scanned images)
2. Using a different PDF viewer to verify text content
3. Converting scanned PDFs to text-based PDFs using OCR

### Large File Errors

Files over 10MB are rejected. Compress or split your documents.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- [Anthropic Claude](https://www.anthropic.com/) for AI vocabulary extraction
- [Charm Bracelet](https://charm.sh/) for the beautiful TUI framework
- [SQLite](https://www.sqlite.org/) for the embedded database
