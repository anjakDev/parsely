# Parsely - Development Guidelines

This document contains guidelines for developing and maintaining Parsely.

## Project Overview

Parsely is a language learning tool that extracts vocabulary from course notes using AI. It follows a security-first, test-driven development approach.

## Architecture Decisions

### Database: SQLite

**Why**: Lightweight, serverless, zero-configuration, perfect for single-user desktop application.

**Alternatives considered**: PostgreSQL (too heavy), MySQL (requires server), JSON files (no querying).

### AI: Claude API

**Why**: State-of-the-art language understanding, JSON output support, reasonable pricing.

**Alternatives considered**: OpenAI GPT (considered), Local models (not accurate enough).

### TUI Framework: Bubbletea

**Why**: Modern, well-maintained, elegant composability, great for interactive CLIs.

**Alternatives considered**: tview (more complex), Survey (less flexible).

### Document Parsers

- **PDF**: `ledongthuc/pdf` - Simple, pure Go, no C dependencies
- **DOCX**: `nguyenthenguyen/docx` - Free, no licensing requirements

**Alternatives considered**: UniOffice (requires license), apache/tika (JVM dependency).

## Code Style Guidelines

### General Principles

1. **Simplicity over cleverness**: Write straightforward, readable code
2. **Security by default**: Validate all inputs, sanitize all outputs
3. **Test-first**: Write tests before implementation
4. **Clear errors**: Error messages should explain what happened and how to fix it
5. **No premature optimization**: Focus on correctness first

### Go Style

Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments):

- Use `gofmt` for formatting
- Keep functions small and focused (< 50 lines when possible)
- Use descriptive variable names
- Comment exported functions
- Return errors, don't panic (except in main)

### File Organization

```
internal/
├── ai/          # AI integration (Claude API)
├── parser/      # Document parsers (PDF, DOCX)
├── db/          # Database layer (SQLite)
├── core/        # Business logic (orchestration)
└── api/         # HTTP handlers (web API)
```

Each package should:
- Have a single, clear responsibility
- Export minimal public API
- Include comprehensive tests
- Document exported functions

## Security Best Practices

### File Upload Security

1. **Validate file extension AND magic bytes**
   ```go
   // Check extension
   if ext != ".pdf" && ext != ".docx" { ... }

   // Validate magic bytes (in parser)
   ```

2. **Limit file size** (10MB max)
   ```go
   const MaxFileSize = 10 * 1024 * 1024
   ```

3. **Sanitize filenames**
   ```go
   // Prevent path traversal
   if strings.Contains(filename, "..") { return error }
   ```

4. **Use temporary files with proper cleanup**
   ```go
   defer parser.CleanupTempFile(tmpPath)
   ```

### Database Security

1. **Always use parameterized queries**
   ```go
   // GOOD
   db.Exec("SELECT * FROM vocabulary WHERE text = ?", userInput)

   // BAD - SQL injection vulnerability
   db.Exec("SELECT * FROM vocabulary WHERE text = '" + userInput + "'")
   ```

2. **Set restrictive file permissions**
   ```go
   os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
   ```

### API Security

1. **CORS**: Only allow specific origins in production
2. **Rate limiting**: Implement on upload endpoint (not yet done - future enhancement)
3. **Input validation**: Validate all request parameters
4. **Error handling**: Don't leak sensitive information in errors

### API Key Management

1. **Never hardcode API keys**
2. **Load from environment variables only**
   ```go
   apiKey := os.Getenv("ANTHROPIC_API_KEY")
   if apiKey == "" { log.Fatal("...") }
   ```
3. **Never log API keys**
4. **Validate format before use**

## Testing Strategy

### Test-Driven Development (TDD)

1. **Write test first**
   - Define expected behavior
   - Write failing test

2. **Implement minimal code**
   - Make test pass
   - Keep it simple

3. **Refactor**
   - Improve code quality
   - Keep tests green

### Test Coverage Goals

- **Minimum**: 50% overall coverage
- **Security functions**: 100% coverage
- **Business logic**: 80% coverage
- **UI code**: Lower priority (harder to test)

### Testing Best Practices

1. **Use table-driven tests** for multiple cases:
   ```go
   tests := []struct {
       input    string
       expected int
   }{
       {"test1", 1},
       {"test2", 2},
   }
   for _, tc := range tests {
       t.Run(tc.input, func(t *testing.T) {
           result := DoThing(tc.input)
           if result != tc.expected { ... }
       })
   }
   ```

2. **Use in-memory database** for tests:
   ```go
   db, _ := db.NewDatabase(":memory:")
   ```

3. **Mock external dependencies**:
   ```go
   type MockAIExtractor struct {
       Vocabulary []string
       Err error
   }
   ```

4. **Test security scenarios**:
   - SQL injection attempts
   - Path traversal attempts
   - Oversized files
   - Malformed input

## Common Patterns

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to parse document: %w", err)
}
```

### Database Transactions

For future enhancement when adding complex operations:
```go
tx, _ := db.Begin()
defer tx.Rollback() // Rollback if not committed

// ... operations ...

tx.Commit()
```

### Graceful Cleanup

```go
defer func() {
    if err := resource.Close(); err != nil {
        log.Printf("cleanup error: %v", err)
    }
}()
```

## Deployment

### Production Checklist

- [ ] Set restrictive CORS origins
- [ ] Enable HTTPS (use reverse proxy like nginx)
- [ ] Set secure database file permissions (0600)
- [ ] Use strong API key
- [ ] Set up log rotation
- [ ] Configure rate limiting
- [ ] Set proper file size limits
- [ ] Test backup/restore procedures

### Environment Variables

```bash
# Production
export ANTHROPIC_API_KEY="sk-ant-..."
export DATABASE_PATH="/var/lib/parsely/parsely.db"
export PORT="8080"
export LANGUAGE="Spanish"
```

### Systemd Service (Linux)

```ini
[Unit]
Description=Parsely Language Learning Tool
After=network.target

[Service]
Type=simple
User=parsely
WorkingDirectory=/opt/parsely
Environment="ANTHROPIC_API_KEY=..."
Environment="DATABASE_PATH=/var/lib/parsely/parsely.db"
ExecStart=/opt/parsely/parsely-web
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Future Enhancements

### High Priority

1. **Rate limiting** on upload endpoint
2. **Batch processing** for multiple files
3. **Context preservation** (store original sentences)
4. **Multi-language support** in UI

### Medium Priority

1. **Anki export** format
2. **User authentication** for web version
3. **Vocabulary categories/tags**
4. **Search functionality**

### Low Priority

1. **Spaced repetition** integration
2. **Translation support**
3. **Audio pronunciation**
4. **Mobile app**

## Troubleshooting Development Issues

### Import Cycles

If you encounter import cycles:
- Ensure `core` doesn't import `api`
- Ensure `db` doesn't import `core`
- Use interfaces to break dependencies

### Test Failures

1. **Race conditions**: Run with `go test -race`
2. **Database locks**: Use `:memory:` or separate DBs per test
3. **File cleanup**: Always use `defer` for cleanup

### Performance Issues

1. **Large files**: Ensure streaming where possible
2. **Database**: Add indexes on frequently queried columns
3. **API**: Implement caching for repeated queries

## Getting Help

- **Documentation**: See README.md for user guide
- **Issues**: Check GitHub issues for known problems
- **Contributing**: Follow CONTRIBUTING.md guidelines

## Changelog

### v1.0.0 (Initial Release)

- AI-powered vocabulary extraction
- PDF and DOCX support
- SQLite database storage
- CLI and web interfaces
- Comprehensive test suite
- Security-first implementation
