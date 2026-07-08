# Tools
.PHONY: default
default: help

UI_SRC := internal/ui/src

## Development:
.PHONY: setup
setup: install-hooks install-ui fmt vet lint ## Setup development environment
	@echo "Development environment setup complete!"

.PHONY: dev
dev: install-ui ## Start the proxy server with live reloading (requires air)
	air

.PHONY: start
start: install-ui ## Start the proxy server
	go run main.go start

.PHONY: build
build: install-ui ## Build the mirra binary
	go build -o mirra .

.PHONY: install
install: install-ui ## Install all dependencies, then build and install mirra to GOBIN (or GOPATH/bin)
	go install .
	@bin_dir="$$(go env GOBIN)"; \
	[ -n "$$bin_dir" ] || bin_dir="$$(go env GOPATH)/bin"; \
	echo "Installed mirra to $$bin_dir/mirra"

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	go test -v ./...

.PHONY: test-race
test-race: ## Run tests with race detector
	go test -race ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage report saved to coverage.out"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

.PHONY: coverage-html
coverage-html: test-coverage ## Generate HTML coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "HTML coverage report saved to coverage.html"
	@which open > /dev/null && open coverage.html || echo "Open coverage.html in your browser"

.PHONY: fmt
fmt: ## Format Go code
	gofmt -s -w .

.PHONY: fmt-check
fmt-check: ## Check if code is formatted (like CI does)
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "gofmt found formatting issues:"; \
		gofmt -s -l .; \
		echo "Run 'make fmt' to fix these issues"; \
		exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: brew install golangci-lint" && exit 1)
	golangci-lint run --timeout=5m ./...

.PHONY: check
check: fmt-check vet lint ## Run all checks (fmt-check, vet, lint) like CI does
	@echo "All checks passed!"

.PHONY: install-ui
install-ui: $(UI_SRC)/node_modules ## Install UI dependencies

# Only reinstall when the package manifest or lockfile changes.
$(UI_SRC)/node_modules: $(UI_SRC)/package.json $(UI_SRC)/package-lock.json
	cd $(UI_SRC) && npm install
	@touch $(UI_SRC)/node_modules

.PHONY: install-hooks
install-hooks: ## Install git hooks
	@bash .githooks/setup.sh

.PHONY: clean
clean: ## Remove built binaries and coverage files
	rm -f mirra coverage.out coverage.html

## Examples
.PHONY: example_prompt
echo_example_prompt: ## Echo the prompt used to build a new example (usage: make echo_example_prompt go openai)
	@sed -e 's/{language}/$(word 1,$(filter-out $@,$(MAKECMDGOALS)))/g' \
	     -e 's/{provider}/$(word 2,$(filter-out $@,$(MAKECMDGOALS)))/g' \
	     _dev/examples_prompt.txt

.PHONY: list_examples
list_examples: ## List available examples
	@echo "Available examples:"
	@for lang in _examples/*/; do \
		if [ -d "$$lang" ]; then \
			echo "$$(basename $$lang):"; \
			for lib in $$lang*/; do \
				if [ -d "$$lib" ]; then \
					echo "  - $$(basename $$lib)"; \
				fi \
			done \
		fi \
	done

.PHONY: run_example
run_example: ## Run an example (usage: make run_example go openai)
	cd _examples/$(word 1,$(filter-out $@,$(MAKECMDGOALS)))/$(word 2,$(filter-out $@,$(MAKECMDGOALS))) && ./run.sh

.PHONY: run_all_examples
run_all_examples: ## Run all examples
	@for lang in _examples/*/; do \
		if [ -d "$$lang" ]; then \
			for lib in $$lang*/; do \
				if [ -d "$$lib" ] && [ -f "$$lib/run.sh" ]; then \
					echo "Running $$(basename $$lang)/$$(basename $$lib)..."; \
					cd "$$lib" && ./run.sh && cd - > /dev/null || exit 1; \
				fi \
			done \
		fi \
	done

.PHONY: chmod_examples
chmod_chmod_examples: ## Make example run scripts executable
	find _examples -name "run.sh" -type f -exec chmod +x {} +


GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)


## Help:
.PHONY: help
help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

%:
	@:
