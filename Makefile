# Tools
.PHONY: default
default: help

## Development:
.PHONY: dev
dev: ## Start the proxy server with live reloading (requires air)
	air

.PHONY: start
start: ## Start the proxy server
	go run main.go start

.PHONY: build
build: ## Build the mirra binary
	go build -o mirra .

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: clean
clean: ## Remove built binaries
	rm -f mirra

## Examples
.PHONY: build_example
build_example: ## Build a new example (usage: make build_example go openai)
	@sed -e 's/{language}/$(word 1,$(filter-out $@,$(MAKECMDGOALS)))/g' \
	     -e 's/{provider}/$(word 2,$(filter-out $@,$(MAKECMDGOALS)))/g' \
	     _dev/examples_prompt.txt | ANTHROPIC_BASE_URL=http://localhost:4567/ claude --dangerously-skip-permissions -p ""

.PHONY: build_example_codex
build_example_codex: ## Build a new example using codex (usage: make build_example_codex go openai)
	@sed -e 's/{language}/$(word 1,$(filter-out $@,$(MAKECMDGOALS)))/g' \
	     -e 's/{provider}/$(word 2,$(filter-out $@,$(MAKECMDGOALS)))/g' \
	     _dev/examples_prompt.txt | CODEX_BASE_URL=http://localhost:4567/ codex --dangerously-skip-permissions -p ""

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
