BINARY  := pvm
DIST    := dist

COMPOSE := docker compose --progress quiet

MAKEFLAGS += -s

HOST_OS   = $(shell docker version --format "{{.Client.Os}}")
HOST_ARCH = $(shell docker version --format "{{.Client.Arch}}")
VM_ARCH   = $(shell docker version --format "{{.Server.Arch}}")

# go_build(goos, goarch, output name): compile in the `go` service into dist/.
define go_build
	@$(COMPOSE) run --rm go sh cli/build $(1) $(2) $(DIST)/$(3)
endef

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@$(COMPOSE) run --rm --no-deps --entrypoint awk go -f cli/help.awk $(firstword $(MAKEFILE_LIST))

.PHONY: setup
setup: ## Build the Docker images
	@$(COMPOSE) build

.PHONY: build
build: setup ## Build for the current OS/arch (output: dist/pvm)
	$(call go_build,$(HOST_OS),$(HOST_ARCH),$(BINARY)$(if $(filter windows,$(HOST_OS)),.exe))

.PHONY: build-all
build-all: setup clean build-linux build-darwin build-windows ## Cross-compile for all target platforms (output: dist/)

.PHONY: build-linux
build-linux: setup
	$(call go_build,linux,amd64,$(BINARY)-linux-amd64)
	$(call go_build,linux,arm64,$(BINARY)-linux-arm64)

.PHONY: build-darwin
build-darwin: setup
	$(call go_build,darwin,amd64,$(BINARY)-darwin-amd64)
	$(call go_build,darwin,arm64,$(BINARY)-darwin-arm64)

.PHONY: build-windows
build-windows: setup
	$(call go_build,windows,amd64,$(BINARY)-windows-amd64.exe)

# pvm binary used by the container; rebuilt only when Go sources change.
# rwildcard(dirs, pattern): recursive $(wildcard), in place of `find`.
rwildcard   = $(foreach d,$(wildcard $(addsuffix /*,$(1))),$(call rwildcard,$(d),$(2)) $(filter $(subst *,%,$(2)),$(d)))
PVM_BIN    := .local/bin/$(BINARY)
GO_SOURCES := main.go go.mod go.sum $(filter-out %_test.go,$(call rwildcard,cmd internal,*.go))

$(PVM_BIN): $(GO_SOURCES)
	@$(COMPOSE) run --rm go sh cli/build linux $(VM_ARCH) $(PVM_BIN)

.PHONY: up
up: setup $(PVM_BIN) ## Start the pvm container in background (state persists until `make down`)
	@$(COMPOSE) up -d app-pvm

.PHONY: down
down: ## Stop and remove the pvm container (resets installed PHP versions)
	@$(COMPOSE) down

.PHONY: pvm
pvm: up ## Run pvm in the container (e.g. make pvm install 8.5, or ARGS="..." to keep quotes)
	@$(COMPOSE) exec app-pvm sh cli/pvm pvm $(PVM_ARGS) $(ARGS)

.PHONY: shell
shell: up ## Open a bash shell in the pvm container
	@$(COMPOSE) exec app-pvm sh cli/pvm bash

# `make pvm run 8.5 teste.php`: words after `pvm` are pvm arguments, not make targets.
ifeq (pvm,$(firstword $(MAKECMDGOALS)))
PVM_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
.PHONY: $(PVM_ARGS)
$(PVM_ARGS): ;
endif

.PHONY: test
test: setup ## Run tests
	@$(COMPOSE) run --rm go go test ./...

.PHONY: lint
lint: setup ## Run go vet
	@$(COMPOSE) run --rm go go vet ./...

.PHONY: clean
clean: setup ## Remove build artifacts
	@$(COMPOSE) run --rm go rm -rf $(DIST)
