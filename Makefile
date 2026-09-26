BINARY  := pvm
MODULE  := github.com/rejmann/pvm
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DIST    := dist

TARGETS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

# Go never runs on the host: every target builds, tests or runs inside Docker (compose.yaml).
HOST_OS   := $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_ARCH := $(subst x86_64,amd64,$(subst aarch64,arm64,$(shell uname -m)))
COMPOSE   := PVM_UID=$(shell id -u) PVM_GID=$(shell id -g) VERSION=$(VERSION) docker compose --progress quiet

# go_build(goos, goarch, output name): compile in the `go` service into the mounted dist/.
define go_build
	@$(COMPOSE) run --rm -e GOOS=$(1) -e GOARCH=$(2) go \
		go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST)/$(3) .
	@echo "  built $(DIST)/$(3)"
endef

.DEFAULT_GOAL := help

.PHONY: help
help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup: ## Build the Docker images (run after changing code or go.mod)
	@mkdir -p $(DIST) .local/bin .local/go-cache .local/apt/archives
	@$(COMPOSE) build

.PHONY: build
build: setup ## Build for the current OS/arch (output: dist/pvm)
	$(call go_build,$(HOST_OS),$(HOST_ARCH),$(BINARY))

.PHONY: build-all
build-all: clean setup build-linux build-darwin build-windows ## Cross-compile for all target platforms (output: dist/)

.PHONY: build-linux
build-linux: setup ## Cross-compile for Linux (amd64 + arm64)
	$(call go_build,linux,amd64,$(BINARY)-linux-amd64)
	$(call go_build,linux,arm64,$(BINARY)-linux-arm64)

.PHONY: build-darwin
build-darwin: setup ## Cross-compile for macOS (amd64 + arm64)
	$(call go_build,darwin,amd64,$(BINARY)-darwin-amd64)
	$(call go_build,darwin,arm64,$(BINARY)-darwin-arm64)

.PHONY: build-windows
build-windows: setup ## Cross-compile for Windows (amd64)
	$(call go_build,windows,amd64,$(BINARY)-windows-amd64.exe)

# pvm binary used by the container; rebuilt only when Go sources change.
PVM_BIN    := .local/bin/$(BINARY)
GO_SOURCES := $(shell find main.go cmd internal -name '*.go' -not -name '*_test.go') go.mod go.sum

$(PVM_BIN): $(GO_SOURCES)
	@$(COMPOSE) run --rm -e GOOS=linux -e GOARCH=$(HOST_ARCH) go \
		go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(PVM_BIN) .

.PHONY: up
up: setup $(PVM_BIN) ## Start the pvm container in background (state persists until `make down`)
	@$(COMPOSE) up -d app-pvm

.PHONY: down
down: ## Stop and remove the pvm container (resets installed PHP versions)
	@$(COMPOSE) down

.PHONY: pvm
pvm: up ## Run pvm in the running container (e.g. make pvm install 8.5, or ARGS="..." to keep quotes)
	@$(COMPOSE) exec app-pvm pvm $(PVM_ARGS) $(ARGS)

.PHONY: shell
shell: up ## Open a bash shell in the pvm container
	@$(COMPOSE) exec app-pvm bash

# `make pvm run 8.5 teste.php`: words after `pvm` are pvm arguments, not make targets.
ifeq (pvm,$(firstword $(MAKECMDGOALS)))
PVM_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
.PHONY: $(PVM_ARGS)
$(PVM_ARGS):
	@:
endif

.PHONY: test
test: setup ## Run tests (in Docker)
	@$(COMPOSE) run --rm go go test ./...

.PHONY: lint
lint: setup ## Run go vet (in Docker)
	@$(COMPOSE) run --rm go go vet ./...

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(DIST)
