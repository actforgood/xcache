LINTER_VERSION=v1.58.1
LINTER=./bin/golangci-lint
ifeq ($(OS),Windows_NT)
	LINTER=./bin/golangci-lint.exe
endif

.PHONY: all
all: clean setup lint test ## Run sequentially clean, setup, lint and test.

.PHONY: lint
lint: ## Run linter and detect go mod tidy changes.
	$(LINTER) run -c ./.golangci-lint.yml --fix
	@make tidy
	@if ! git diff --quiet; then \
		echo "'go mod tidy' resulted in changes or working tree is dirty:"; \
		git --no-pager diff; \
	fi

.PHONY: setup
setup: ## Download dependencies.
	go mod download
	@if [ ! -f "$(LINTER)" ]; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(LINTER_VERSION); \
	fi

.PHONY: test
test: ## Run tests (with race condition detection).
	go test -race -timeout=30s ./...

.PHONY: test-integration
test-integration: ## Run integration tests (with race condition detection).
	go test -race -timeout=5m -tags=integration ./...	

.PHONY: bench
bench: ## Run benchmarks.
	go test -race -benchmem -benchtime=5s -bench=.

.PHONY: bench-integration
bench-integration: ## Run integration benchmarks.
	go test -race -tags=integration -benchmem -benchtime=5s -bench=.

.PHONY: cover
cover: ## Run tests with coverage. Generates "cover.out" profile and its html representation.
	go test -race -timeout=30s -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

.PHONY: cover-integration
cover-integration: ## Run integration tests with coverage. Generates "cover-integration.out" profile and its html representation.
	go test -race -timeout=5m -tags=integration -coverprofile=cover-integration.out -coverpkg=./... ./...
	go tool cover -html=cover-integration.out -o cover-integration.html

.PHONY: tidy
tidy: ## Simply runs 'go mod tidy'.
	go mod tidy

.PHONY: clean
clean: ## Clean up go tests cache and coverage generated files.
	go clean -testcache
	@for file in cover.html cover.out cover-integration.html cover-integration.out; do \
		if [ -f $$file ]; then \
			echo "rm -f $$file"; \
			rm -f $$file; \
		fi \
	done

.PHONY: help
# Absolutely awesome: https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
