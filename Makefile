BINARY := cocopilot
BUILD_DIR := dist
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64
VERSION := $(shell cat VERSION 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/onsomlem/cocopilot/server.Version=$(VERSION)"
CMD := ./cmd/cocopilot

.PHONY: build build-all test test-coverage bench clean lint run docker-build docker-run help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build binary for current platform
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY) $(CMD)

build-all: ## Cross-compile for all platforms
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output=$(BUILD_DIR)/$(BINARY)-$${os}-$${arch}; \
		echo "Building $${os}/$${arch} -> $${output}"; \
		CGO_ENABLED=0 GOOS=$${os} GOARCH=$${arch} go build $(LDFLAGS) -o $${output} $(CMD) || exit 1; \
	done

run: build ## Build and run the server
	./$(BINARY)

test: ## Run tests with race detection
	go test -race -timeout 180s ./...

test-coverage: ## Run tests and show coverage report
	go test -coverprofile=coverage.out -timeout 180s ./server/
	go tool cover -func=coverage.out | tail -1
	@echo "Run 'go tool cover -html=coverage.out' for detailed HTML report"

bench: ## Run benchmarks
	go test -bench . -benchtime 5x -timeout 60s ./server/

clean: ## Remove build artifacts
	rm -f $(BINARY) coverage.out
	rm -rf $(BUILD_DIR)

lint: ## Run go vet
	go vet ./...

docker-build: ## Build Docker image
	docker build -t cocopilot:$(VERSION) .

docker-run: docker-build ## Build and run in Docker
	docker run --rm -p 8080:8080 -v cocopilot-data:/data cocopilot:$(VERSION)
