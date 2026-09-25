BINARY  := pvm
MODULE  := github.com/rejmann/pvm
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X $(MODULE)/cmd.version=$(VERSION)"
DIST    := dist

TARGETS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.DEFAULT_GOAL := help

.PHONY: help
help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build for the current OS/arch (output: dist/)
	@mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/$(BINARY) .

.PHONY: build-all
build-all: ## Cross-compile for all target platforms (output: dist/)
	@mkdir -p $(DIST)
	$(foreach TARGET,$(TARGETS), \
		$(eval OS   := $(word 1,$(subst /, ,$(TARGET)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(TARGET)))) \
		$(eval OUT  := $(DIST)/$(BINARY)-$(OS)-$(ARCH)$(if $(filter windows,$(OS)),.exe,)) \
		GOOS=$(OS) GOARCH=$(ARCH) go build $(LDFLAGS) -o $(OUT) . && \
		echo "  built $(OUT)" ; \
	)

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

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: lint
lint: ## Run go vet
	go vet ./...

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(DIST)
