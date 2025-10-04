# Taco - LLM API Proxy Specification

## Overview

Taco is a transparent HTTP proxy for Large Language Model APIs (Claude and OpenAI) that records all request/response traffic without modifying it. It acts as a pass-through intermediary, allowing developers to inspect, audit, and analyze their LLM API usage.

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

## Architecture

```
Client � Taco Proxy � Upstream API (Claude/OpenAI)
              �
         Recording Store
```

### Components

1. **HTTP Server**: Listens for incoming requests
2. **Router**: Matches requests to appropriate upstream API
3. **Proxy Handler**: Forwards requests and pipes responses
4. **Recorder**: Asynchronously records request/response data
5. **Storage**: Persists recorded data

## Request Flow

1. Client sends request to Taco (e.g., `http://localhost:4567/v1/messages`)
2. Taco identifies the target API from configuration
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
  "provider": "claude|openai",
  "request": {
    "method": "POST",
    "path": "/v1/messages",
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
- The full reconstructed response is stored
- Streaming flag is set to `true`
- Chunks are passed through immediately without buffering

## Configuration

Configuration via JSON file or environment variables:

```json
{
  "port": 4567,
  "recording": {
    "enabled": true,
    "storage": "file", // we will add more storage options in the future
    "path": "./recordings",
    "format": "jsonl"
  },
  "logging": {
    "format": "pretty",
    "level": "info"
  },
  "providers": {
    "claude": {
      "upstream_url": "https://api.anthropic.com",
    },
    "openai": {
      "upstream_url": "https://api.openai.com",
    }
  }
}
```

### Environment Variables

- `TACO_PORT` - Server port (default: 4567)
- `TACO_RECORDING_ENABLED` - Enable/disable recording (default: true)
- `TACO_RECORDING_PATH` - Path to store recordings (default: ./recordings)
- `TACO_CLAUDE_UPSTREAM` - Claude upstream URL
- `TACO_OPENAI_UPSTREAM` - OpenAI upstream URL

## CLI Commands

### Start Server

```bash
taco start [--port 4567] [--config ./config.json]
```

Starts the proxy server.

### Export Recordings

```bash
taco export [--from 2025-10-01] [--to 2025-10-03] [--provider claude|openai] [--output recordings.jsonl]
```

Exports recorded traffic to a file.

### Stats

```bash
taco stats [--from 2025-10-01] [--provider claude|openai]
```

Shows statistics about recorded traffic:
- Total requests
- Average response time
- Token usage (if available in responses)
- Error rate

### View Recording

```bash
taco view <recording-id>
```

Displays a specific recording in formatted output.

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
- Optional: Redact sensitive fields from recordings
- Support for TLS/HTTPS

### Observability
- Health check endpoint: `GET /health`

### Logging

Taco uses structured logging (Go's `log/slog` package) with three configurable output formats:

**Pretty Format (default)**: Custom human-readable handler with:
- Color-coded log levels with visual symbols (▪ debug, ◆ info, ▲ warn, ● error)
- Special formatting for proxy request logs with taco symbol ⛁
- Provider-specific colored symbols (◆ claude in orange, ● openai in green)
- Human-readable duration formatting (ms/s/m)
- Status code color coding (green 2xx, blue 3xx, yellow 4xx, red 5xx)
- Request log format: `[TIME] ⛁ [ID] [PROVIDER] [DURATION] [STATUS] [PATH]`

**JSON Format**: Standard structured JSON output for log aggregation systems

**Plain Format**: Standard text format with key=value pairs

Example pretty request log:
```
[14:23:45] ⛁ a1b2c3d4 ◆ claude 1.2s 200 /v1/messages
[14:23:46] ◆ info taco proxy server started port=4567 recording=true
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
