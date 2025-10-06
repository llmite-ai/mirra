# 𝕄𝕀ℝℝ𝔸

**M**onitoring & **I**nspection **R**ecording **R**elay **A**rchive

A transparent HTTP proxy for Large Language Model APIs that records all request/response traffic without modifying it.

MIRRA acts as a pass-through intermediary for inspection, auditing, and analysis of LLM API usage. Currently supports Claude (Anthropic), OpenAI, and Google Gemini APIs.

## Features

- **Transparent proxying**: Requests and responses pass through unmodified
- **Multi-provider support**: Claude (Anthropic), OpenAI, and Google Gemini APIs
- **Streaming support**: Handles both regular and Server-Sent Events (SSE) streaming responses
- **Asynchronous recording**: Records traffic without adding latency to API calls
- **Compression handling**: Automatically handles and records gzip-compressed responses
- **Export & analysis**: Built-in commands to export and analyze recorded traffic
- **Advanced viewing**: Partial UUID matching, automatic redaction of sensitive data, SSE formatting
- **Structured logging**: Multiple output formats (pretty, JSON, plain) with color-coded request logs

## Installation

### Prerequisites

- Go 1.21 or later

### Build from source

```bash
go build -o mirra .
```

## Usage

### Start the proxy server

```bash
./mirra start
```

By default, the proxy listens on port `4567`. You can specify a custom port:

```bash
./mirra start --port 8080
```

Or provide a configuration file:

```bash
./mirra start --config ./config.json
```

### Configure your API client

Point your LLM API client to the MIRRA proxy instead of the upstream API:

**Claude:**
```bash
# Instead of: https://api.anthropic.com
# Use: http://localhost:4567
```

**OpenAI:**
```bash
# Instead of: https://api.openai.com
# Use: http://localhost:4567
```

**Gemini:**
```bash
# Instead of: https://generativelanguage.googleapis.com
# Use: http://localhost:4567
```

Keep your API keys unchanged - MIRRA forwards them to the upstream APIs.

### Export recordings

Export all recordings:

```bash
./mirra export --output traffic.jsonl
```

Export with filters:

```bash
./mirra export --from 2025-01-01 --to 2025-01-31 --provider claude --output claude-jan.jsonl
```

Options:
- `--from` - Start date (YYYY-MM-DD)
- `--to` - End date (YYYY-MM-DD)
- `--provider` - Filter by provider (claude, openai, or gemini)
- `--output` - Output file path (default: export.jsonl)
- `--recordings` - Path to recordings directory (default: ./recordings)

### View statistics

```bash
./mirra stats
```

Filter by date range or provider:

```bash
./mirra stats --from 2025-01-01 --provider openai
```

Options:
- `--from` - Start date (YYYY-MM-DD)
- `--provider` - Filter by provider (claude, openai, or gemini)
- `--recordings` - Path to recordings directory (default: ./recordings)

### View a specific recording

View a recording by ID (supports partial UUID matching):

```bash
./mirra view a1b2c3d4
```

View the most recent recording:

```bash
./mirra view
```

Features:
- Partial UUID matching - just provide the first few characters
- Automatically redacts sensitive data (API keys, tokens)
- Decompresses gzip-compressed responses
- Formats streaming SSE responses for readability
- Pretty-prints JSON

Options:
- `<recording-id>` - Full or partial UUID (optional, defaults to last recording)
- `--recordings` - Path to recordings directory (default: ./recordings)

## Configuration

Configuration can be provided via a JSON file or environment variables.

### Configuration file (config.json)

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

### Environment variables

Environment variables override config file values:

- `MIRRA_PORT` - Server port (default: 4567)
- `MIRRA_RECORDING_ENABLED` - Enable/disable recording (default: true)
- `MIRRA_RECORDING_PATH` - Directory for recording files (default: ./recordings)
- `MIRRA_CLAUDE_UPSTREAM` - Claude API upstream URL
- `MIRRA_OPENAI_UPSTREAM` - OpenAI API upstream URL
- `MIRRA_GEMINI_UPSTREAM` - Gemini API upstream URL

### Logging

MIRRA supports three logging formats via the `logging.format` configuration:

- **pretty** (default): Human-readable with color-coded log levels and request symbols
- **json**: Structured JSON output for log aggregation systems
- **plain**: Standard text format with key=value pairs

Log levels: `debug`, `info`, `warn`, `error`

## Recording Format

Recordings are stored as JSONL files (one JSON object per line) with the naming pattern `recordings-YYYY-MM-DD.jsonl`.

Each recording includes:

```json
{
  "id": "uuid-v4",
  "timestamp": "2025-01-15T10:30:00Z",
  "provider": "claude|openai|gemini",
  "request": {
    "method": "POST",
    "path": "/v1/messages",
    "query": "key=value",
    "headers": {
      "content-type": ["application/json"]
    },
    "body": {}
  },
  "response": {
    "status": 200,
    "headers": {
      "content-type": ["application/json"]
    },
    "body": {},
    "streaming": false
  },
  "timing": {
    "started_at": "2025-01-15T10:30:00.123Z",
    "completed_at": "2025-01-15T10:30:02.456Z",
    "duration_ms": 2333
  }
}
```

**Note**: For gzip-compressed responses, the body is stored as base64-encoded with a "base64:" prefix.

## Supported API Endpoints

### Claude (Anthropic)
- `/v1/messages` - Messages API (streaming and non-streaming)
- `/v1/complete` - Legacy completion API

### OpenAI
- `/v1/chat/completions` - Chat completions (streaming and non-streaming)
- `/v1/completions` - Legacy completions
- `/v1/embeddings` - Embeddings
- `/v1/models` - List models
- `/v1/models/:id` - Retrieve model
- `/v1/responses` - Responses API

### Gemini (Google)
All Gemini API endpoints across versions (v1, v1beta, v1alpha):
- Model operations (generateContent, streamGenerateContent, embedContent, countTokens, etc.)
- File operations (upload, list, get, delete)
- Cached contents management
- Corpora and semantic retrieval (documents, chunks)
- Tuned models (operations, permissions)
- Batch operations

Example endpoints:
- `/v1/models/gemini-pro:generateContent`
- `/v1beta/models/gemini-2.5-pro:streamGenerateContent`
- `/v1/files`
- `/upload/v1/files`

## Examples

### Using with curl

**OpenAI:**
```bash
# Start the proxy
./mirra start

# Make a request through the proxy
curl -X POST http://localhost:4567/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Claude:**
```bash
curl -X POST http://localhost:4567/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: YOUR_ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 1024
  }'
```

**Gemini:**
```bash
curl -X POST "http://localhost:4567/v1/models/gemini-2.5-pro:generateContent?key=YOUR_GEMINI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "parts": [{"text": "Hello!"}]
    }]
  }'
```

The request and response will be automatically recorded in the `./recordings` directory.

## Performance

MIRRA is designed for minimal overhead:
- Target latency: < 1ms additional overhead
- Streaming responses pass through in real-time
- Recording happens asynchronously without blocking requests

## License

See LICENSE file for details.
