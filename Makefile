.PHONY: test-ci test-ci-build

test-ci-build:
	docker build -t gmail-mcp-ci:local -f ./Dockerfile.ci ./

test-ci: test-ci-build
	docker run --rm -v ./:/workspace gmail-mcp-ci:local go test -mod=vendor -v -race ./...
