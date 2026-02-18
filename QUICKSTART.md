# Parsely Quick Start Guide

Get started with Parsely in 5 minutes!

## Step 1: Get Your API Key

1. Visit [Anthropic Console](https://console.anthropic.com/)
2. Sign up or log in
3. Navigate to API Keys
4. Create a new API key
5. Copy the key (starts with `sk-ant-`)

## Step 2: Set Up Environment

```bash
# Set your API key
export ANTHROPIC_API_KEY="sk-ant-your-key-here"

# Optional: Customize settings
export DATABASE_PATH="my_vocabulary.db"
export LANGUAGE="Spanish"
```

Or create a `.env` file (copy from `.env.example`):

```bash
cp .env.example .env
# Edit .env with your favorite editor
nano .env
```

Then load it:
```bash
source .env
```

## Step 3: Choose Your Interface

### Option A: CLI (Terminal UI)

```bash
# Run the CLI
./parsely-cli

# Navigate with:
# - Arrow keys or j/k (vim style)
# - Enter to select
# - q to quit or go back
```

**What you can do:**
1. Parse new document - Select a PDF or DOCX file
2. View all vocabulary - Browse extracted vocabulary
3. Export to JSON - Save vocabulary to a file
4. Exit - Close the application

### Option B: Web API

```bash
# Start the web server
./parsely-web

# Server starts at http://localhost:8080
```

**Try it out:**

```bash
# Upload a document
curl -X POST -F "file=@spanish_lesson.pdf" http://localhost:8080/api/upload

# View all vocabulary
curl http://localhost:8080/api/vocabulary

# Get stats
curl http://localhost:8080/api/stats

# Export to JSON
curl -O http://localhost:8080/api/export
```

## Step 4: Process Your First Document

### Using CLI:

1. Launch `./parsely-cli`
2. Select "Parse new document"
3. Enter the full path to your PDF or DOCX file
4. Wait for processing (uses Claude AI)
5. View results showing new vocabulary added

### Using API:

```bash
curl -X POST \
  -F "file=@/path/to/your/document.pdf" \
  http://localhost:8080/api/upload

# Response:
{
  "new_vocabulary": 25,
  "skipped_duplicates": 5,
  "total_processed": 30,
  "language": "Spanish"
}
```

## Step 5: View Your Vocabulary

### CLI:
1. Select "View all vocabulary" from the main menu
2. Browse through all extracted vocabulary
3. Press Enter to return to menu

### API:
```bash
# Get all vocabulary
curl http://localhost:8080/api/vocabulary | jq

# Get specific item
curl http://localhost:8080/api/vocabulary/1

# Export to file
curl http://localhost:8080/api/export > vocabulary.json
```

## Common Use Cases

### Batch Processing Multiple Documents

```bash
# Using API
for file in lessons/*.pdf; do
  echo "Processing $file..."
  curl -X POST -F "file=@$file" http://localhost:8080/api/upload
  sleep 2  # Rate limiting
done
```

### Export for Anki Flashcards

```bash
# Export to JSON
curl http://localhost:8080/api/export > vocabulary.json

# Convert to Anki CSV (you'll need to write a simple script)
# Format: Front,Back
# Example: "hola","hello"
```

### Search Vocabulary

```bash
# Get all vocabulary and search with jq
curl http://localhost:8080/api/vocabulary | jq '.[] | select(.text | contains("hola"))'
```

### Delete Vocabulary

```bash
# Delete by ID
curl -X DELETE http://localhost:8080/api/vocabulary/5
```

## Tips & Tricks

### 1. Organize by Language

Set the language for each processing session:
```bash
export LANGUAGE="French"
./parsely-cli
# Process French documents

export LANGUAGE="Spanish"
./parsely-cli
# Process Spanish documents
```

### 2. Backup Your Database

```bash
# Create backup
cp parsely.db parsely_backup_$(date +%Y%m%d).db

# Restore from backup
cp parsely_backup_20260218.db parsely.db
```

### 3. Use Multiple Databases

```bash
# Spanish vocabulary
DATABASE_PATH="spanish.db" LANGUAGE="Spanish" ./parsely-web

# French vocabulary (different terminal)
DATABASE_PATH="french.db" LANGUAGE="French" ./parsely-web
```

### 4. Monitor Processing

Watch the web server logs:
```bash
./parsely-web | tee parsely.log
```

### 5. Test with Sample Document

Create a simple test document:

**test_spanish.txt** (save as PDF):
```
Lección 1: Saludos

Hola - Hello
Buenos días - Good morning
Buenas tardes - Good afternoon
Buenas noches - Good evening
¿Cómo estás? - How are you?
Muy bien, gracias - Very well, thank you
```

## Troubleshooting

### Problem: "ANTHROPIC_API_KEY not set"
**Solution**: Export your API key:
```bash
export ANTHROPIC_API_KEY="your-key"
```

### Problem: "File too large"
**Solution**: Files must be under 10MB. Compress or split your document.

### Problem: "Unsupported file type"
**Solution**: Only PDF and DOCX files are supported. Convert other formats first.

### Problem: "No vocabulary extracted"
**Solution**:
- Ensure the document contains readable text (not scanned images)
- Check that the text is in the target language
- Verify the PDF has selectable text

### Problem: Database locked
**Solution**: Close other instances of Parsely using the same database.

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Check [CLAUDE.md](CLAUDE.md) for development guidelines
- Report issues on GitHub
- Contribute improvements!

## Need Help?

- **API Documentation**: See README.md API Endpoints section
- **Development Guide**: See CLAUDE.md
- **Security**: See CLAUDE.md Security section
- **Issues**: Report on GitHub

Happy vocabulary learning! 📚
