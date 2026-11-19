# memctl — build and verify targets.
#
#   make build    # bin/memctl for this machine (ADR-0012)
#   make verify   # test + vet + fmt-check + check-docs (workflow Phase 4)
#
# Release binaries for linux/amd64, linux/arm64 and darwin/arm64 are built by
# the GitHub Actions release workflow, started by hand for an existing vX.Y.Z
# tag, not here (ADR-0012).

BINARY  := memctl
MAIN    := ./cmd/memctl
BIN_DIR := bin

# Version injected into main.version via -ldflags. Only vX.Y.Z tags count; the
# leading "v" is stripped so the MCP server info reports "X.Y.Z", the same
# string as the CHANGELOG heading.
VERSION ?= $(shell (git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null || echo dev) | sed 's/^v//')
LDFLAGS := -s -w -X main.version=$(VERSION)

# The host platform by default; override for a one-off build, e.g.
#   make build GOOS=linux GOARCH=amd64
GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.DEFAULT_GOAL := help

.PHONY: build
build: ## Build bin/memctl for this machine (CGO_ENABLED=0)
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(MAIN)

.PHONY: test
test: ## Run the full test suite
	go test ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: fmt
fmt: ## Format all Go files in place
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## Fail if any Go file is not gofmt-clean
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed on:"; echo "$$unformatted"; exit 1; \
	fi

.PHONY: check-docs
check-docs: ## Check the spec and ADR indexes and the agent-facing tool text
	bash scripts/check-docs.sh

.PHONY: verify
verify: test vet fmt-check check-docs ## Full verification gate (workflow Phase 4)

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
