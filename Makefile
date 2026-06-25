# Makefile for qris-go. Run `make help` for the available targets.

GO    ?= go
PKG   ?= ./...

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile all packages.
	$(GO) build $(PKG)

.PHONY: test
test: ## Run the test suite.
	$(GO) test $(PKG)

.PHONY: test-race
test-race: ## Run the test suite with the race detector.
	$(GO) test -race $(PKG)

.PHONY: cover
cover: ## Run tests and report coverage.
	$(GO) test -coverprofile=coverage.out $(PKG)
	$(GO) tool cover -func=coverage.out

.PHONY: cover-html
cover-html: cover ## Generate an HTML coverage report.
	$(GO) tool cover -html=coverage.out -o coverage.html

.PHONY: vet
vet: ## Run go vet.
	$(GO) vet $(PKG)

.PHONY: lint
lint: ## Run golangci-lint (must be installed).
	golangci-lint run $(PKG)

.PHONY: fmt
fmt: ## Format the code with gofmt.
	gofmt -w .

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum.
	$(GO) mod tidy

.PHONY: ci
ci: build vet test-race lint ## Run everything CI runs.

.PHONY: clean
clean: ## Remove build and coverage artifacts.
	rm -f coverage.out coverage.html gencrc gentestdata
