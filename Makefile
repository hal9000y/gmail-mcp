.PHONY: test-ci test-ci-build codeql codeql-db codeql-analyze

test-ci-build:
	docker build -t gmail-mcp-ci:local -f ./Dockerfile.ci ./

test-ci: test-ci-build
	docker run --rm -v ./:/workspace gmail-mcp-ci:local go test -v -race ./...

codeql-db:
	@echo "Building CodeQL database..."
	codeql database create ./build/codeql \
		--language=go \
		--overwrite \
		--command 'go build ./...'

codeql-analyze:
	@echo "Running CodeQL analysis..."
	codeql database analyze ./build/codeql \
		codeql-pack/go-security-and-quality.qls \
		--format=csv \
		--output=./build/codeql-results.csv
	@echo "Results saved to ./build/codeql-results.csv"

codeql: codeql-db codeql-analyze
