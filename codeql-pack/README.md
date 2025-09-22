# CodeQL Pack for Gmail MCP

Minimal CodeQL configuration for security analysis of the Gmail MCP Go codebase.

## Usage

### Create Database

```bash
# Create CodeQL database with explicit build command (recommended)
codeql database create ./build/codeql \
  --language=go \
  --overwrite \
  --command 'go build ./...'
```

**Note for macOS ARM:** Always use explicit `--command` flag as autodetect may fail on ARM architecture.

### Run Analysis

```bash
# Run analysis using database at ./build/codeql
codeql database analyze ./build/codeql codeql-pack/go-security-and-quality.qls \
  --format=sarif-latest --output=./build/codeql-results.sarif

# Generate CSV report
codeql database analyze ./build/codeql codeql-pack/go-security-and-quality.qls \
  --format=csv --output=./build/codeql-results.csv

# Update existing database after code changes
codeql database upgrade ./build/codeql
```

## Query Suite

Currently using the standard `go-security-and-quality` suite from CodeQL, which includes:
- Security vulnerabilities (CWE coverage)
- Code quality issues
- Best practices violations
