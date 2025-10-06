# MIRRA - LLM API Proxy Specification

**M**onitoring & **I**nspection **R**ecording **R**elay **A**rchive

## Overview

MIRRA is a transparent HTTP proxy for Large Language Model APIs (Claude, OpenAI, and Gemini) that records all request/response traffic without modifying it. It acts as a pass-through intermediary, allowing developers to inspect, audit, and analyze their LLM API usage.

## Design Principles

1. **Transparency**: The proxy must be completely transparent - requests and responses pass through unmodified
2. **Non-blocking**: Recording failures must not affect the request/response flow
3. **Compatibility**: Routes must exactly match the upstream API specifications
4. **Observability**: All traffic is recorded with timing and metadata

## Supported APIs

### Claude (Anthropic)
- `POST /v1/messages` - Messages API (streaming and non-streaming)
- `POST /v1/complete` - Legacy completion API
- `GET /v1/messages/:id` - Retrieve message (if available)

### OpenAI
- `POST /v1/chat/completions` - Chat completions (streaming and non-streaming)
- `POST /v1/completions` - Legacy completions
- `POST /v1/embeddings` - Embeddings
- `GET /v1/models` - List models
- `GET /v1/models/:id` - Retrieve model
- `POST /v1/responses` - Responses API

### Gemini (Google)
- Model operations: `/v1*/models/*` (generateContent, streamGenerateContent, embedContent, countTokens, etc.)
- File operations: `/v1*/files`, `/v1*/files/*`, `/upload/v1*/files`
- Cached contents: `/v1*/cachedContents`, `/v1*/cachedContents/*`
- Corpora and semantic retrieval: `/v1*/corpora`, `/v1*/corpora/*` (includes documents and chunks)
- Tuned models: `/v1*/tunedModels`, `/v1*/tunedModels/*` (includes operations and permissions)
- Batch operations: `/v1*/batches`, `/v1*/batches/*`

Supports API versions: v1, v1beta, v1alpha

## Architecture

```
Client → MIRRA Proxy → Upstream API (Claude/OpenAI/Gemini)
              ↓
         Recording Store
```

### Components

1. **HTTP Server**: Listens for incoming requests
2. **Router**: Matches requests to appropriate upstream API
3. **Proxy Handler**: Forwards requests and pipes responses
4. **Recorder**: Asynchronously records request/response data
5. **Storage**: Persists recorded data

## Request Flow

1. Client sends request to MIRRA (e.g., `http://localhost:4567/v1/messages`)
2. MIRRA identifies the target API from configuration
3. Request body and headers are captured
4. Request is forwarded to upstream API
5. Response is streamed back to client in real-time
6. Simultaneously, response is captured for recording
7. Recording is persisted asynchronously (failures logged but not propagated)

## Recording Format

Each request/response pair is recorded as a JSON document:

```json
{
  "id": "uuid-v4",
  "timestamp": "2025-10-03T20:52:00Z",
  "provider": "claude|openai|gemini",
  "request": {
    "method": "POST",
    "path": "/v1/messages",
    "query": "key=value&param=data",
    "headers": {
      "content-type": "application/json",
      "anthropic-version": "2023-06-01"
    },
    "body": { /* original request body */ }
  },
  "response": {
    "status": 200,
    "headers": {
      "content-type": "application/json"
    },
    "body": { /* original response body */ },
    "streaming": false
  },
  "timing": {
    "started_at": "2025-10-03T20:52:00.123Z",
    "completed_at": "2025-10-03T20:52:02.456Z",
    "duration_ms": 2333
  }
}
```

### Streaming Handling

For streaming responses (SSE):
- Each chunk is accumulated and recorded
- The full reconstructed response is stored as a string (in SSE format)
- Streaming flag is set to `true`
- Chunks are passed through immediately without buffering

### Compression Handling

For gzip-compressed responses:
- Response body is base64-encoded with "base64:" prefix to preserve binary data
- The `view` command automatically decompresses and displays the content

## Configuration

Configuration via JSON file or environment variables:

```json
{
  "port": 4567,
  "recording": {
    "enabled": true,
    "storage": "file",
    "path": "./recordings",
    "format": "jsonl"
  },
  "logging": {
    "format": "pretty",
    "level": "info"
  },
  "providers": {
    "claude": {
      "upstream_url": "https://api.anthropic.com"
    },
    "openai": {
      "upstream_url": "https://api.openai.com"
    },
    "gemini": {
      "upstream_url": "https://generativelanguage.googleapis.com"
    }
  }
}
```

### Environment Variables

- `MIRRA_PORT` - Server port (default: 4567)
- `MIRRA_RECORDING_ENABLED` - Enable/disable recording (default: true)
- `MIRRA_RECORDING_PATH` - Path to store recordings (default: ./recordings)
- `MIRRA_CLAUDE_UPSTREAM` - Claude upstream URL
- `MIRRA_OPENAI_UPSTREAM` - OpenAI upstream URL
- `MIRRA_GEMINI_UPSTREAM` - Gemini upstream URL

## CLI Commands

### Start Server

```bash
mirra start [--port 4567] [--config ./config.json]
```

Starts the proxy server.

### Export Recordings

```bash
mirra export [--from 2025-10-01] [--to 2025-10-03] [--provider claude|openai|gemini] [--output recordings.jsonl] [--recordings ./recordings]
```

Exports recorded traffic to a file.

Options:
- `--from` - Start date (YYYY-MM-DD)
- `--to` - End date (YYYY-MM-DD)
- `--provider` - Filter by provider (claude, openai, or gemini)
- `--output` - Output file path (default: export.jsonl)
- `--recordings` - Path to recordings directory (default: ./recordings)

### Stats

```bash
mirra stats [--from 2025-10-01] [--provider claude|openai|gemini] [--recordings ./recordings]
```

Shows statistics about recorded traffic:
- Total requests
- Average response time
- Error rate
- Per-provider breakdown

Options:
- `--from` - Start date (YYYY-MM-DD)
- `--provider` - Filter by provider (claude, openai, or gemini)
- `--recordings` - Path to recordings directory (default: ./recordings)

### View Recording

```bash
mirra view [recording-id] [--recordings ./recordings]
```

Displays a specific recording in formatted output.

Features:
- Supports partial UUID matching (e.g., `mirra view a1b2c3d4` to match full UUID `a1b2c3d4-...`)
- If no recording ID provided, shows the last/most recent recording
- Automatically redacts sensitive data (API keys, tokens) from headers and query parameters
- Automatically decompresses and formats gzip-compressed responses
- Special formatting for streaming SSE responses with event-by-event breakdown
- Pretty-prints JSON request and response bodies

Options:
- `<recording-id>` - Full or partial UUID of the recording to view (optional, defaults to last recording)
- `--recordings` - Path to recordings directory (default: ./recordings)

## Storage Options

### File System (JSONL)
- One line per request/response pair
- File rotation by date: `recordings-2025-10-03.jsonl`
- Simple, portable, grep-able

## Technical Requirements

### Performance
- Minimal latency overhead (< 1ms)
- Non-blocking recording (async writes)
- Efficient memory usage for streaming responses

### Error Handling
- Upstream API errors are passed through transparently
- Recording errors are logged but don't affect client
- Connection failures to upstream result in 502 Bad Gateway

### Security
- Sensitive data redaction in `view` command:
  - Authorization and X-Api-Key headers are redacted
  - Query parameters (key, apiKey, api_key, token, access_token) are redacted
- Support for TLS/HTTPS

### Observability
- Health check endpoint: `GET /health`

### Logging

MIRRA uses structured logging (Go's `log/slog` package) with three configurable output formats:

**Pretty Format (default)**: Custom human-readable handler with:
- Color-coded log levels with visual symbols (▪ debug, ◆ info, ▲ warn, ● error)
- Special formatting for proxy request logs with proxy symbol ⛁
- Provider-specific colored symbols (◆ claude in orange, ● openai in green)
- Human-readable duration formatting (ms/s/m)
- Status code color coding (green 2xx, blue 3xx, yellow 4xx, red 5xx)
- Request log format: `[TIME] ⛁ [ID] [PROVIDER] [DURATION] [STATUS] [PATH]`

**JSON Format**: Standard structured JSON output for log aggregation systems

**Plain Format**: Standard text format with key=value pairs

Example pretty request log:
```
[14:23:45] ⛁ a1b2c3d4 ◆ claude 1.2s 200 /v1/messages
[14:23:46] ◆ info MIRRA proxy server started port=4567 recording=true
```

## Future Enhancements (don't implement yet)

- Store in sqlite and postgres
- Request replay from recordings
- Request/response transformation hooks
- Rate limiting
- Caching layer
- Multiple upstream endpoints (load balancing)
- Web UI for browsing recordings
- Real-time streaming of recordings (WebSocket)
- Request filtering (by path, headers, etc.)
- Cost tracking and budgets
- Alerting on errors or usage patterns
- Other LLM providers
