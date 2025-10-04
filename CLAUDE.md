# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Taco is a transparent HTTP proxy for Large Language Model APIs (Claude and OpenAI) that records all request/response traffic without modifying it. The proxy acts as a pass-through intermediary for inspection, auditing, and analysis of LLM API usage.

## Commands

### Building
```bash
go build -o taco .
```

### Running the server
```bash
./taco start [--port 4567] [--config ./config.json]
```

Default port is 4567. Configuration can be provided via JSON file or environment variables.

### Running commands
```bash
./taco export [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--provider claude|openai] [--output file.jsonl]
./taco stats [--from YYYY-MM-DD] [--provider claude|openai]
./taco view <recording-id>
```

### Testing
No test suite currently exists.

## Architecture

The codebase follows a clean separation of concerns:

- **main.go**: CLI entry point with command routing (start, export, stats, view)
- **internal/server**: HTTP server setup and health check endpoint
- **internal/proxy**: Core proxy logic that handles request forwarding and response streaming
- **internal/recorder**: Asynchronous recording of request/response pairs to JSONL files
- **internal/config**: Configuration loading from files and environment variables
- **internal/commands**: CLI command implementations (export, stats, view)

### Request Flow
1. Client sends request to Taco proxy (e.g., `http://localhost:4567/v1/messages`)
2. Proxy identifies provider (Claude or OpenAI) from URL path
3. Request body and headers are captured
4. Request is forwarded to upstream API
5. Response is streamed back to client in real-time
6. Response is captured simultaneously
7. Recording is persisted asynchronously (failures logged but not propagated to client)

### Provider Routing
Provider identification happens in `proxy.identifyProvider()` based on URL path prefixes:
- Claude: `/v1/messages`, `/v1/complete`
- OpenAI: `/v1/chat/completions`, `/v1/completions`, `/v1/embeddings`, `/v1/models`

### Recording Format
Recordings are stored as JSONL files (one JSON object per line) in the format `recordings-YYYY-MM-DD.jsonl`. Each recording includes:
- Unique ID (UUID v4)
- Provider (claude/openai)
- Request data (method, path, headers, body)
- Response data (status, headers, body, streaming flag)
- Timing data (start time, end time, duration in ms)

### Streaming Handling
The proxy supports both regular and streaming (SSE) responses:
- Streaming responses are detected via `Content-Type: text/event-stream`
- Chunks are passed through immediately to client without buffering
- Full response is accumulated for recording
- Uses `http.Flusher` interface for real-time streaming

### Configuration
Default configuration:
- Port: 4567
- Recording enabled: true
- Recording path: `./recordings`
- Claude upstream: `https://api.anthropic.com`
- OpenAI upstream: `https://api.openai.com`

Environment variables override config file values:
- `TACO_PORT`
- `TACO_RECORDING_ENABLED`
- `TACO_RECORDING_PATH`
- `TACO_CLAUDE_UPSTREAM`
- `TACO_OPENAI_UPSTREAM`

### Async Recording
The recorder uses a buffered channel (capacity 100) with a background worker goroutine to write recordings asynchronously. If the channel is full, recordings are dropped with a warning. On shutdown, the recorder drains remaining recordings before closing.

## Design Principles

1. **Transparency**: Requests and responses pass through unmodified
2. **Non-blocking**: Recording failures must not affect request/response flow
3. **Compatibility**: Routes exactly match upstream API specifications
4. **Observability**: All traffic recorded with timing and metadata
5. **Minimal latency**: Target < 1ms overhead
