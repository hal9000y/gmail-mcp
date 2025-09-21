# CodeQL Pack for Gmail MCP

Minimal CodeQL configuration for security analysis of the Gmail MCP Go codebase.

## Structure

```
codeql-pack/
├── qlpack.yml                 # Pack metadata and dependencies
├── codeql-config.yml          # Analysis configuration
├── go-security-and-quality.qls # Query suite import
└── README.md                  # This file
```

## Usage

### Create Database

```bash
# Create CodeQL database with explicit build command (recommended)
codeql database create ./build/codeql \
  --language=go \
  --overwrite \
  --command 'go build ./...'
```

**Note for macOS ARM (M1/M2):** Always use explicit `--command` flag as autodetect may fail on ARM architecture.

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

## Resources

### Official Documentation
- [CodeQL Documentation](https://codeql.github.com/docs/)
- [Creating CodeQL packs](https://docs.github.com/en/code-security/codeql-cli/using-the-advanced-functionality-of-the-codeql-cli/creating-codeql-query-packs)
- [CodeQL for Go](https://codeql.github.com/docs/codeql-language-guides/codeql-for-go/)

### Query Development
- [Writing CodeQL queries](https://codeql.github.com/docs/writing-codeql-queries/)
- [CodeQL query help for Go](https://codeql.github.com/codeql-query-help/go/)
- [CodeQL standard library for Go](https://codeql.github.com/codeql-standard-libraries/go/)

### Available Query Suites
- [Go query suites](https://github.com/github/codeql/tree/main/go/ql/src/codeql-suites)
- [Security queries](https://github.com/github/codeql-go/tree/main/ql/src/Security)
- [CWE coverage](https://codeql.github.com/codeql-query-help/go/#cwe-coverage)

## Customization

To add custom queries:
1. Create `queries/` directory
2. Add `.ql` files with your queries
3. Create a custom suite file that includes them
4. Reference in `qlpack.yml` dependencies

## GitHub Actions Integration

```yaml
- name: Initialize CodeQL
  uses: github/codeql-action/init@v2
  with:
    config-file: ./codeql-pack/codeql-config.yml
    queries: ./codeql-pack/go-security-and-quality.qls
```