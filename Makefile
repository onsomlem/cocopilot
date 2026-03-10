BINARY := cocopilot
BUILD_DIR := dist
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64
VERSION := $(shell cat VERSION 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/onsomlem/cocopilot/server.Version=$(VERSION)"
CMD := ./cmd/cocopilot

.PHONY: build build-all test test-coverage bench clean lint run docker-build docker-run release verify-release verify-repo verify-source gate fresh-test help

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

test-unit: ## Run unit tests only
	go test -run "TestUnit" -timeout 30s ./server/

test-e2e: ## Run e2e tests only
	go test -run "TestE2E|TestScannerE2E" -timeout 60s ./server/

test-smoke: ## Run smoke tests for all routes
	go test -run "TestSmoke" -timeout 30s ./server/

test-contract: ## Run API contract tests
	go test -run "TestContract" -timeout 30s ./server/

test-integration: ## Run integration lifecycle tests
	go test -run "TestIntegration" -timeout 60s ./server/

test-coverage: ## Run tests and show coverage report
	go test -coverprofile=coverage.out -timeout 180s ./server/
	go tool cover -func=coverage.out | tail -1
	@echo "Run 'go tool cover -html=coverage.out' for detailed HTML report"

test-ci: build test ## CI pipeline: build, test with race detection, coverage gate
	@go test -coverprofile=coverage.out -timeout 180s ./server/ > /dev/null 2>&1; \
	total=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | tr -d '%'); \
	echo "Coverage: $${total}%"; \
	threshold=65; \
	if [ $$(echo "$${total} < $${threshold}" | bc) -eq 1 ]; then \
		echo "FAIL: coverage $${total}% is below $${threshold}% threshold"; \
		exit 1; \
	fi; \
	echo "CI: All checks passed"

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

release: ## Build and package a clean release zip
	bash scripts/package.sh

verify-release: ## Validate release zip is clean
	@echo "=== Verifying release zip ===" && \
	VERIFY_DIR=$$(mktemp -d) && \
	if [ ! -f $(BUILD_DIR)/cocopilot-release.zip ]; then \
		echo "No release zip found. Run 'make release' first."; \
		rm -rf "$$VERIFY_DIR"; \
		exit 1; \
	fi && \
	unzip -q $(BUILD_DIR)/cocopilot-release.zip -d "$$VERIFY_DIR" && \
	FAILED=0 && \
	if find "$$VERIFY_DIR" -type d -name ".git" | grep -q .; then \
		echo "FAIL: .git/ directory found in release"; FAILED=1; \
	fi && \
	if find "$$VERIFY_DIR" -name "tasks.db*" | grep -q .; then \
		echo "FAIL: database files found in release"; FAILED=1; \
	fi && \
	if find "$$VERIFY_DIR" -type d -name "__MACOSX" | grep -q .; then \
		echo "FAIL: __MACOSX found in release"; FAILED=1; \
	fi && \
	if find "$$VERIFY_DIR" -name ".DS_Store" | grep -q .; then \
		echo "FAIL: .DS_Store found in release"; FAILED=1; \
	fi && \
	if find "$$VERIFY_DIR" -name "*.exe~" | grep -q .; then \
		echo "FAIL: stray binary artifacts found in release"; FAILED=1; \
	fi && \
	if find "$$VERIFY_DIR" -name "coverage.out" | grep -q .; then \
		echo "FAIL: coverage.out found in release"; FAILED=1; \
	fi && \
	if find "$$VERIFY_DIR" -type d -name "node_modules" | grep -q .; then \
		echo "FAIL: node_modules found in release"; FAILED=1; \
	fi && \
	if find "$$VERIFY_DIR" -name "*.vsix" | grep -q .; then \
		echo "FAIL: .vsix binaries found in release"; FAILED=1; \
	fi && \
	FILE_COUNT=$$(find "$$VERIFY_DIR" -type f | wc -l | tr -d ' ') && \
	ZIP_SIZE=$$(du -sh $(BUILD_DIR)/cocopilot-release.zip | cut -f1) && \
	rm -rf "$$VERIFY_DIR" && \
	if [ "$$FAILED" -eq 1 ]; then \
		echo "FAIL: Release zip contains forbidden artifacts"; \
		exit 1; \
	fi && \
	echo "OK: Release zip clean ($$FILE_COUNT files, $$ZIP_SIZE)"

verify-repo: ## Check git index for banned artifacts
	@echo "=== Verifying repo cleanliness ===" && \
	FAILED=0 && \
	if git ls-files | grep -qE '\.vsix$$'; then \
		echo "FAIL: .vsix files tracked in git"; FAILED=1; \
	fi && \
	if git ls-files | grep -qE 'node_modules/'; then \
		echo "FAIL: node_modules tracked in git"; FAILED=1; \
	fi && \
	if git ls-files | xargs file 2>/dev/null | grep -q 'Mach-O\|ELF\|PE32'; then \
		echo "FAIL: compiled binaries tracked in git"; FAILED=1; \
	fi && \
	if git ls-files | grep -qE '\.db$$|\.db-shm$$|\.db-wal$$'; then \
		echo "FAIL: database files tracked in git"; FAILED=1; \
	fi && \
	if [ "$$FAILED" -eq 1 ]; then \
		echo "FAIL: Repo contains banned artifacts"; \
		exit 1; \
	fi && \
	echo "OK: Repo is clean"

verify-source: ## Check working tree for forbidden artifacts
	@echo "=== Verifying source tree ===" && \
	FAILED=0 && \
	if find . -maxdepth 1 -name "*.db" -o -name "*.db-shm" -o -name "*.db-wal" | grep -q .; then \
		echo "FAIL: database files in repo root"; FAILED=1; \
	fi && \
	if find . -maxdepth 1 -name "*.zip" | grep -q .; then \
		echo "FAIL: zip archives in repo root"; FAILED=1; \
	fi && \
	if find . -maxdepth 1 -name "*.exe~" | grep -q .; then \
		echo "FAIL: stale binary artifacts in repo root"; FAILED=1; \
	fi && \
	if [ -f ./cocopilot ] && file ./cocopilot | grep -qE 'Mach-O|ELF|PE32'; then \
		echo "FAIL: compiled binary 'cocopilot' in repo root"; FAILED=1; \
	fi && \
	if find . -type d -name "__MACOSX" | grep -q .; then \
		echo "FAIL: __MACOSX directories in tree"; FAILED=1; \
	fi && \
	if find . -name ".DS_Store" | grep -q .; then \
		echo "FAIL: .DS_Store files in tree"; FAILED=1; \
	fi && \
	if find . -name "coverage.out" | grep -q .; then \
		echo "FAIL: coverage.out in tree"; FAILED=1; \
	fi && \
	if [ "$$FAILED" -eq 1 ]; then \
		echo "FAIL: Source tree contains forbidden artifacts"; \
		echo "Hint: run 'make clean' or remove files manually"; \
		exit 1; \
	fi && \
	echo "OK: Source tree is clean"

gate: ## Hard release gate: all checks must pass before shipping
	@echo "=== Release Gate ===" && \
	echo "[1/6] Repo cleanliness (git index)..." && $(MAKE) -s verify-repo && \
	echo "[2/6] Source tree cleanliness..." && $(MAKE) -s verify-source && \
	echo "[3/6] Lint..." && $(MAKE) -s lint && \
	echo "[4/6] Build..." && $(MAKE) -s build && \
	echo "[5/6] Tests (race + golden path)..." && go test -race -timeout 180s ./... && \
	echo "[6/6] Verify release..." && $(MAKE) -s release && $(MAKE) -s verify-release && \
	echo "" && echo "=== ALL GATES PASSED ==="

fresh-test: ## Clone into temp dir and validate build+test+package from scratch
	bash scripts/fresh-machine-test.sh
