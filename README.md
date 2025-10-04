# Taco

A transparent HTTP proxy for Large Language Model APIs that records all request/response traffic without modifying it.

Taco acts as a pass-through intermediary for inspection, auditing, and analysis of LLM API usage. Currently supports Claude (Anthropic) and OpenAI APIs.

## Features

- **Transparent proxying**: Requests and responses pass through unmodified
- **Streaming support**: Handles both regular and Server-Sent Events (SSE) streaming responses
- **Asynchronous recording**: Records traffic without adding latency to API calls
- **Multi-provider**: Supports Claude and OpenAI APIs
- **Export & analysis**: Built-in commands to export and analyze recorded traffic

## Installation

### Prerequisites

- Go 1.21 or later

### Build from source

```bash
go build -o taco .
```

## Usage

### Start the proxy server

```bash
./taco start
```

By default, the proxy listens on port `4567`. You can specify a custom port:

```bash
./taco start --port 8080
```

Or provide a configuration file:

```bash
./taco start --config ./config.json
```

### Configure your API client

Point your LLM API client to the Taco proxy instead of the upstream API:

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

Keep your API keys unchanged - Taco forwards them to the upstream APIs.

### Export recordings

Export all recordings:

```bash
./taco export --output traffic.jsonl
```

Export with filters:

```bash
./taco export --from 2025-01-01 --to 2025-01-31 --provider claude --output claude-jan.jsonl
```

### View statistics

```bash
./taco stats
```

Filter by date range or provider:

```bash
./taco stats --from 2025-01-01 --provider openai
```

### View a specific recording

```bash
./taco view <recording-id>
```

## Configuration

Configuration can be provided via a JSON file or environment variables.

### Configuration file (config.json)

```json
{
  "port": 4567,
  "recordingEnabled": true,
  "recordingPath": "./recordings",
  "claudeUpstream": "https://api.anthropic.com",
  "openaiUpstream": "https://api.openai.com"
}
```

### Environment variables

Environment variables override config file values:

- `TACO_PORT` - Server port (default: 4567)
- `TACO_RECORDING_ENABLED` - Enable/disable recording (default: true)
- `TACO_RECORDING_PATH` - Directory for recording files (default: ./recordings)
- `TACO_CLAUDE_UPSTREAM` - Claude API upstream URL
- `TACO_OPENAI_UPSTREAM` - OpenAI API upstream URL

## Recording Format

Recordings are stored as JSONL files (one JSON object per line) with the naming pattern `recordings-YYYY-MM-DD.jsonl`.

Each recording includes:

```json
{
  "id": "uuid-v4",
  "provider": "claude|openai",
  "request": {
    "method": "POST",
    "path": "/v1/messages",
    "headers": {},
    "body": {}
  },
  "response": {
    "status": 200,
    "headers": {},
    "body": {},
    "streaming": false
  },
  "timing": {
    "startTime": "2025-01-15T10:30:00Z",
    "endTime": "2025-01-15T10:30:02Z",
    "durationMs": 2000
  }
}
```

## Supported API Endpoints

### Claude (Anthropic)
- `/v1/messages`
- `/v1/complete`

### OpenAI
- `/v1/chat/completions`
- `/v1/completions`
- `/v1/embeddings`
- `/v1/models`

## Example: Using with curl

```bash
# Start the proxy
./taco start

# Make a request through the proxy
curl -X POST http://localhost:4567/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

The request and response will be automatically recorded in the `./recordings` directory.

## Performance

Taco is designed for minimal overhead:
- Target latency: < 1ms additional overhead
- Streaming responses pass through in real-time
- Recording happens asynchronously without blocking requests

## License

See LICENSE file for details.
