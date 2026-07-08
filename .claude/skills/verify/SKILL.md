---
name: verify
description: Build and drive the mirra proxy end-to-end to verify a change — launch it against fake or real upstreams, send traffic through it, and inspect logs plus recordings JSONL.
---

# Verifying mirra changes

mirra is an HTTP(S)+WebSocket recording proxy. Verifying a change means
sending real traffic through a running instance and reading what it
recorded, never just unit tests.

## Launch

```bash
go build -o /tmp/mirra-test .   # or any scratch path
MIRRA_PORT=4599 \
MIRRA_RECORDING_PATH=./recordings \
LOG_FORMAT=plain \
/tmp/mirra-test start > mirra.log 2>&1 &
curl -s http://127.0.0.1:4599/health   # "OK" when up
```

Gotchas:

- The user usually has a live instance on **4567 — do not kill or reuse
  it**. Pick another port. `pkill -f "./mirra start"` only matches test
  instances started that way; the installed one runs as plain `mirra`.
- A stale test instance holds the port and the new one exits with
  `bind: address already in use` while its log still opens normally —
  check `lsof -iTCP:<port> -sTCP:LISTEN` and the log tail after starting.
- Point providers at fake upstreams via env:
  `MIRRA_CLAUDE_UPSTREAM`, `MIRRA_OPENAI_UPSTREAM`,
  `MIRRA_GEMINI_UPSTREAM`, `MIRRA_CHATGPT_UPSTREAM` (codex
  subscription traffic; upstream path prefix `/backend-api/codex`).

## Drive

- Plain HTTP: `curl http://127.0.0.1:4599/v1/...` — unknown API paths
  must 404 with "unknown API endpoint"; `GET /` must serve the UI HTML.
- WebSocket (codex transport, `GET /v1/responses` + `Upgrade: websocket`
  + `ChatGPT-Account-ID` header): no common CLI speaks raw RFC 6455
  here; hand-roll a Go client/upstream (see
  `internal/proxy/websocket_test.go` for frame helpers to copy).
- Real codex end-to-end (needs codex login):
  `codex -c openai_base_url="http://localhost:4599/v1" exec "Reply with exactly: pong"`
  Expect `websocket tunnel established` in the log and 101 recordings.

## Observe

- Log: `request completed` lines carry provider/status/duration.
- Recordings: `recordings/recordings-YYYY-MM-DD.jsonl`, one JSON object
  per line; websocket recordings have `response.status == 101` and
  `response.body` as a list of `{direction, at, type, data}` messages.
  A websocket recording is only written when the connection closes.
