# Repository Guidelines

## Project Structure & Module Organization
`main.go` wires the CLI to command handlers in `internal/commands`. Proxy runtime code is grouped under `internal/server`, `internal/proxy`, and `internal/recorder`; configuration and logging helpers live in `internal/config` and `internal/logger`. Persisted traffic resides in `recordings/` (one JSONL per day). Use `_dev/` for local pokes or fixtures and avoid merging ad-hoc helpers from that folder.

## Build, Run, and Development Commands
`go build -o taco .` produces the binary in the repo root. `make start` or `go run main.go start` launches the proxy on port 4567, while `./taco start --config ./config.json` loads custom settings. `make dev` (Air required) enables live reload, and `make clean` removes cached binaries. Power users expose recordings via `./taco export`, `./taco stats`, and `./taco view`; document example invocations when behavior changes.

## Coding Style & Naming Conventions
Format Go code with `gofmt` or `goimports` before committing; tabs are standard and lines should stay comfortably under 120 characters. Keep package names lowercase and descriptive, exported API in PascalCase, and internals camelCase. Prefer structured logs through `internal/logger` rather than `fmt.Printf`, and reuse existing helper types when extending recorder payloads.

## Testing Guidelines
`go test ./...` is the canonical entry even though coverage is light today. Add table-driven tests in `_test.go` files beside the code, naming functions `TestThing_Scenario`. Capture streaming or gzip fixtures under `_dev/` or temporary directories, and record any manual verification steps in the PR body until automated coverage exists.

## Commit & Pull Request Guidelines
Follow the Conventional Commit style seen in history (`feat:`, `fix:`, `chore:`) and keep logical changes isolated by package. Pull requests should state the user-facing impact, list build or CLI commands exercised, and reference linked issues. Include logs or screenshots only when they illuminate behavior, and scrub customer-identifying details.

## Security & Configuration Tips
Never commit live API keys; rely on `TACO_*` environment variables or an ignored `config.json`. Validate new upstream URLs or headers in both `internal/config` and `internal/proxy` to preserve transparency. When expanding recordings, confirm JSONL output remains gzip-safe and free of unnecessary PII.
