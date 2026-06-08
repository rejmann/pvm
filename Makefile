BINARY  := pvm
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/rejmann/pvm/cmd.version=$(VERSION)"

.DEFAULT_GOAL := help

.PHONY: help
help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build for the current OS/arch
	go build $(LDFLAGS) -o $(BINARY) .

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: lint
lint: ## Run go vet
	go vet ./...

.PHONY: clean
clean: ## Remove build artifacts
	rm -f $(BINARY) $(BINARY).exe
