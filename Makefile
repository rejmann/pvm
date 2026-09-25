BINARY  := pvm
MODULE  := github.com/rejmann/pvm
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
DIST    := dist

TARGETS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

HOST_OS    := $(shell go env GOOS)
HOST_ARCH  := $(shell go env GOARCH)
HOST_BIN   := $(DIST)/$(BINARY)-$(HOST_OS)-$(HOST_ARCH)$(if $(filter windows,$(HOST_OS)),.exe,)
GO_SOURCES := $(shell find . -name '*.go' -not -name '*_test.go' -not -path './$(DIST)/*') go.mod go.sum

.DEFAULT_GOAL := help

.PHONY: help
help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build for the current OS/arch (output: dist/)
	@mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/$(BINARY) .

.PHONY: build-all
build-all: clean build-linux build-darwin build-windows ## Cross-compile for all target platforms (output: dist/)

.PHONY: build-linux
build-linux: ## Cross-compile for Linux (amd64 + arm64)
	@mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-arm64 .

.PHONY: build-darwin
build-darwin: ## Cross-compile for macOS (amd64 + arm64)
	@mkdir -p $(DIST)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-arm64 .

.PHONY: build-windows
build-windows: ## Cross-compile for Windows (amd64)
	@mkdir -p $(DIST)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-windows-amd64.exe .

$(HOST_BIN): $(GO_SOURCES)
	@mkdir -p $(DIST)
	GOOS=$(HOST_OS) GOARCH=$(HOST_ARCH) go build $(LDFLAGS) -o $@ .

.PHONY: run
run: $(HOST_BIN) ## Run the dist/ binary for the current OS/arch (e.g. make run ARGS="list")
	@./$(HOST_BIN) $(ARGS)

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: lint
lint: ## Run go vet
	go vet ./...

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(DIST)
