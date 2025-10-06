# Tools
.PHONY: default help
default: help


GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

## Development:
dev: ## Start the proxy server with live reloading (requires air)
	air

start: ## Start the proxy server
	go run main.go start

build: ## Build the mirra binary
	go build -o mirra .

test: ## Run tests
	go test ./...

clean: ## Remove built binaries
	rm -f mirra

## Pokes

poke_gemini: ## Poke the Gemini server
	(cd _dev/gemini && go run main.go)

## Help:
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
