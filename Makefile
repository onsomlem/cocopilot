BINARY := cocopilot
BUILD_DIR := dist
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64
VERSION := $(shell cat VERSION 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/onsomlem/cocopilot/server.Version=$(VERSION)"
CMD := ./cmd/cocopilot

.PHONY: build build-all test clean lint

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY) $(CMD)

build-all:
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output=$(BUILD_DIR)/$(BINARY)-$${os}-$${arch}; \
		echo "Building $${os}/$${arch} -> $${output}"; \
		CGO_ENABLED=0 GOOS=$${os} GOARCH=$${arch} go build $(LDFLAGS) -o $${output} $(CMD) || exit 1; \
	done

test:
	go test -race -timeout 180s ./...

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)

lint:
	go vet ./...
