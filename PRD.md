# MIRRA Web UI - Product Requirements Document

## Executive Summary

MIRRA currently provides CLI-based tools for viewing and analyzing LLM API traffic. This PRD outlines a web-based interface that brings real-time monitoring, intuitive browsing, and powerful analysis capabilities to developers debugging LLM applications, teams tracking API usage, and security personnel auditing traffic.

## Vision

Transform MIRRA from a CLI-only proxy into a comprehensive observability platform with a web interface that feels like a fusion of Chrome DevTools, Postman, and DataDog APM - purpose-built for LLM API traffic.

## Goals

1. **Real-time Visibility**: Stream live API traffic as it flows through the proxy
2. **Effortless Debugging**: Instantly inspect any request/response with formatted, searchable views
3. **Historical Analysis**: Browse, search, and filter thousands of recordings efficiently
4. **Performance Insights**: Visualize latency, error rates, and usage patterns over time
5. **Zero Friction**: No setup, no build process - just start MIRRA and open a browser

## Non-Goals (v1)

- User authentication/authorization
- Multi-user collaboration features
- Request modification or replay (future enhancement)
- Database storage (uses existing JSONL files)
- Mobile-optimized UI
- Request transformation or mocking

## User Personas

### 1. Application Developer (Primary)
**Sarah** is building an LLM-powered chatbot and needs to debug why certain prompts fail.
- Wants to see exact request/response payloads
- Needs to understand why streaming responses break
- Must verify her application sends correct headers/parameters
- Values fast iteration and minimal context switching

### 2. Platform Engineer (Secondary)
**Marcus** manages LLM integrations for a product team.
- Monitors API health and performance across providers
- Tracks costs and usage patterns
- Needs to identify performance bottlenecks
- Creates reports for stakeholders

### 3. Security/Compliance Lead (Secondary)
**Priya** ensures API usage meets security standards.
- Audits what data is sent to LLM providers
- Verifies sensitive data is properly redacted
- Needs to review historical traffic for compliance
- Creates audit trails for security reviews

## Key Features

### 1. Live Activity Stream

**Description**: Real-time display of API requests as they flow through the proxy.

**UI Elements**:
- Full-width timeline view with newest requests at top
- Auto-scroll toggle for continuous monitoring
- Compact row per request showing:
  - Timestamp (relative: "2s ago" or absolute: "14:32:05")
  - Provider badge (colored: Claude orange, OpenAI green, Gemini blue)
  - HTTP method and path
  - Status code (color-coded: 2xx green, 4xx yellow, 5xx red)
  - Duration (with threshold highlighting: >2s yellow, >5s red)
  - Token count (if extractable from response)
  - Streaming indicator (animated for in-progress)
- Click any row to open Request Inspector
- Keyboard navigation (↑/↓ arrows, Enter to open)

**Technical Requirements**:
- WebSocket connection to `/ws/live` endpoint
- Buffer last 100 requests in memory for instant display
- Graceful reconnection on connection loss
- CPU-efficient rendering (virtual scrolling or incremental DOM updates)

### 2. Request Inspector (Detail View)

**Description**: Comprehensive view of a single request/response pair.

**Layout**:
- Drawer slides in from right (80% viewport width)
- OR modal overlay (for smaller screens)
- Close with ESC key or X button

**Tabs**:

#### Overview Tab
- Request summary card:
  - Provider, model, endpoint
  - Request time, duration, status
  - Token usage (input/output if available)
  - Estimated cost (based on known pricing)
- Quick actions:
  - Copy request as cURL command
  - Copy request body
  - Copy response body
  - Share link (generates shareable URL with recording ID)
  - Download as JSON

#### Request Tab
- Method and full path with query parameters
- Headers section (collapsible, redacted sensitive values)
- Body section:
  - Syntax-highlighted JSON
  - Collapsible nested objects
  - Search within body
  - Copy button for entire body or selected text
  - Line numbers

#### Response Tab
- Status code and status text
- Headers section (collapsible)
- Body section:
  - Auto-detected format (JSON, SSE, plain text)
  - For JSON: Syntax-highlighted, collapsible
  - For SSE: Event-by-event breakdown with timestamps
  - For gzip: Auto-decompressed
  - Search within body
  - Copy button

#### Timing Tab
- Timeline visualization:
  - Request queued
  - DNS lookup (if measurable)
  - Connection established
  - TLS handshake (if measurable)
  - Request sent
  - Waiting for response (TTFB)
  - Response received
- Waterfall chart showing duration breakdown
- Comparison to average latency for this endpoint/model

#### Metadata Tab
- Recording ID (full UUID, copyable)
- Timestamp (ISO 8601 and human-readable)
- Recording file path
- Raw JSON option (show entire recording object)

**Technical Requirements**:
- GET `/api/recordings/{id}` endpoint
- Lazy load tabs (only fetch data when tab opened)
- Efficient JSON rendering (don't render huge payloads at once)
- Syntax highlighting library (lightweight, e.g., Prism.js subset)

### 3. Search & Filtering

**Description**: Powerful filtering to find specific requests in historical data.

**Filter Bar** (sticky at top):
- **Text search**: Full-text search across path, request body, response body
- **Provider**: Multi-select (Claude, OpenAI, Gemini, All)
- **Date range**: Date picker (from/to) with presets (Today, Last 7 days, Last 30 days)
- **Status**: Dropdown (All, 2xx, 4xx, 5xx, or specific codes)
- **Model**: Dropdown (populated from actual recordings)
- **Endpoint**: Dropdown (popular endpoints)
- **Duration**: Range slider (0-10s)
- **Streaming**: Checkbox (streaming only, non-streaming only, all)

**Filter UI**:
- Inline filter bar with dropdowns and inputs
- "Advanced filters" toggle for additional options
- Clear all filters button
- Active filters shown as removable tags
- Filter count badge ("3 filters active")

**URL State**:
- All filters encoded in URL query params
- Shareable filtered views
- Browser back/forward navigation

**Saved Filters** (v1.1):
- Save current filters as named preset
- Quick access dropdown of saved filters
- Stored in localStorage

**Technical Requirements**:
- Efficient query of JSONL files (streaming read, line-by-line parse)
- Pagination (load 50-100 at a time, infinite scroll)
- Debounced text search (300ms delay)
- Background worker for heavy queries (Web Worker)

### 4. Dashboard & Analytics

**Description**: High-level overview of API usage and performance.

**Layout**: Grid of cards and charts

**Metrics Cards** (top row):
- Total requests (with time range)
- Total tokens (estimate)
- Average latency
- Error rate (%)
- Cost estimate (if pricing data available)

**Charts**:
1. **Request Volume Over Time**
   - Line or bar chart
   - Group by hour/day/week based on time range
   - Stacked by provider
   - Hover for exact count

2. **Latency Percentiles**
   - Line chart (p50, p95, p99)
   - X-axis: time
   - Identify latency spikes

3. **Status Code Distribution**
   - Pie or donut chart
   - 2xx, 4xx, 5xx breakdown
   - Click to filter

4. **Provider Breakdown**
   - Bar chart or table
   - Requests, tokens, cost, avg latency per provider
   - Sort by any column

5. **Top Models**
   - Table showing most-used models
   - Request count, tokens, avg latency

6. **Top Endpoints**
   - Table showing most-hit endpoints
   - Request count, error rate, avg latency

7. **Recent Errors**
   - List of last 10-20 errors
   - Click to open Request Inspector

**Time Range Selector**:
- Dropdown: Last hour, 24 hours, 7 days, 30 days, All time, Custom range
- Applies to all charts and metrics

**Technical Requirements**:
- GET `/api/stats?from=&to=&provider=` endpoint
- Efficient aggregation (consider caching daily stats)
- Lightweight charting library (Chart.js or custom SVG)
- Responsive layout (cards stack on smaller screens)

### 5. Settings Panel

**Description**: Configure UI preferences and proxy settings.

**Sections**:

#### Appearance
- Theme: Auto, Light, Dark
- Density: Comfortable, Compact
- Font size: Small, Medium, Large

#### Behavior
- Auto-scroll activity stream: On/Off
- Activity stream limit: 50/100/200/Unlimited
- WebSocket auto-reconnect: On/Off
- Refresh interval (for polling fallback): 5s/10s/30s

#### Recording Path
- Display current recording path
- Link to open in file browser (if possible)

#### Proxy Status
- Show proxy running status
- Port number
- Recording enabled/disabled
- Link to view config

**Technical Requirements**:
- Settings stored in localStorage
- GET `/api/config` endpoint (read-only view of proxy config)
- GET `/api/health` endpoint (proxy status)

## UI Design Principles

### Visual Design

**Color Palette**:
- Background: Dark (#1a1a1a) with light mode option (#ffffff)
- Surface: Cards/panels (#252525 dark, #f5f5f5 light)
- Primary: Blue (#3b82f6)
- Success: Green (#22c55e)
- Warning: Yellow (#eab308)
- Error: Red (#ef4444)
- Text: High contrast (white on dark, black on light)

**Typography**:
- System font stack: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif
- Monospace (code): "SF Mono", Monaco, "Cascadia Code", monospace
- Base size: 14px
- Headers: 18px, 16px

**Provider Colors**:
- Claude: Orange (#ff6b35)
- OpenAI: Green (#10a37f)
- Gemini: Blue (#4285f4)

**Status Colors**:
- 2xx: Green (#22c55e)
- 3xx: Blue (#3b82f6)
- 4xx: Yellow (#eab308)
- 5xx: Red (#ef4444)

**Icons**:
- Use Unicode symbols where possible (⛁, ◆, ●, ▲)
- Fallback to minimal SVG icons

### Interaction Patterns

**Keyboard Shortcuts**:
- `/` - Focus search
- `Esc` - Close inspector/modal
- `↑/↓` - Navigate list
- `Enter` - Open selected item
- `⌘K` or `Ctrl+K` - Command palette (v1.1)
- `r` - Refresh/reload
- `?` - Show keyboard shortcuts help

**Responsive Behavior**:
- Designed for desktop (1280px+ optimal)
- Tablet: Stacked layout, drawer becomes full-screen modal
- Mobile: Not optimized (future enhancement)

**Loading States**:
- Skeleton screens for initial load
- Inline spinners for updates
- Progress bars for long operations (export, large queries)

**Empty States**:
- "No recordings yet" with setup instructions
- "No results" with suggestion to adjust filters
- "Connection lost" with reconnect button

## Technical Architecture

### Backend (Go)

**New Endpoints**:

```
GET  /ui/                          → Serve web UI (embedded HTML/CSS/JS)
GET  /api/recordings               → List recordings (paginated, filtered)
GET  /api/recordings/:id           → Get single recording
GET  /api/stats                    → Get aggregated stats
GET  /api/config                   → Get proxy config (read-only)
GET  /api/health                   → Health check
GET  /ws/live                      → WebSocket for live streaming
```

**Query Parameters** (for `/api/recordings`):
- `?limit=50` - Page size (default 50, max 200)
- `?offset=0` - Pagination offset
- `?from=2025-01-01` - Start date (YYYY-MM-DD)
- `?to=2025-01-31` - End date
- `?provider=claude` - Filter by provider
- `?status=200` - Filter by status code
- `?search=hello` - Full-text search
- `?streaming=true` - Filter streaming
- `?sort=timestamp_desc` - Sort order

**Response Format** (for `/api/recordings`):
```json
{
  "recordings": [
    { /* full recording object */ }
  ],
  "total": 1234,
  "limit": 50,
  "offset": 0,
  "has_more": true
}
```

**WebSocket Protocol**:
- Client connects to `/ws/live`
- Server pushes JSON message for each new recording:
  ```json
  {
    "type": "new_recording",
    "data": { /* full recording object */ }
  }
  ```
- Heartbeat every 30s to keep connection alive
- Client can send `{"type":"ping"}`, server responds `{"type":"pong"}`

**Implementation Details**:
- Embed UI assets using Go 1.16+ `embed` package
- Serve from `/ui/` prefix to avoid conflict with proxy routes
- CORS headers for local development
- Middleware to add `/ui/` route group
- Reuse existing recorder types and file reading logic
- WebSocket using `gorilla/websocket` or `nhooyr.io/websocket`
- Broadcast channel for new recordings

### Frontend (Vanilla JS + CSS)

**File Structure**:
```
web/
  index.html           → Main HTML
  css/
    styles.css         → Main stylesheet
    syntax.css         → Syntax highlighting
  js/
    main.js            → App initialization
    api.js             → API client
    websocket.js       → WebSocket handler
    components/
      activity.js      → Activity stream
      inspector.js     → Request inspector
      dashboard.js     → Dashboard & charts
      filters.js       → Filter bar
      settings.js      → Settings panel
    utils/
      format.js        → Date, number formatting
      highlight.js     → Syntax highlighting
      theme.js         → Theme management
```

**No Build Process**:
- ES6 modules (`<script type="module">`)
- Native CSS variables for theming
- No transpilation, no bundling
- Works in modern browsers (Chrome 90+, Firefox 88+, Safari 14+)

**State Management**:
- Simple event emitter pattern
- Local state in components
- localStorage for preferences
- URL state for filters

**Dependencies** (Minimal):
- None required - use native browser APIs where possible
- Optional: Lightweight chart library (<10KB gzipped)
- Optional: Date formatting (day.js <3KB)

**Performance Optimizations**:
- Virtual scrolling for activity list (only render visible rows)
- Lazy load inspector tabs
- Debounce search input
- Request deduplication
- Efficient JSON rendering (progressive disclosure)
- Web Worker for heavy filtering/searching

## User Flows

### Flow 1: First-Time Setup and Exploration

1. User starts MIRRA: `./mirra start`
2. CLI outputs: "Proxy running at http://localhost:4567 | Web UI at http://localhost:4567/ui/"
3. User opens browser to http://localhost:4567/ui/
4. Empty state shown: "No recordings yet. Make an API request through the proxy to see it here."
5. User makes first API request through proxy
6. Request appears in activity stream in real-time
7. User clicks row to open inspector
8. Inspector shows formatted request/response
9. User closes inspector, sees request in list

### Flow 2: Debugging a Failing Request

1. User's app makes failing API request (500 error)
2. Request appears in activity stream with red status code
3. User clicks to open inspector
4. User navigates to "Response" tab
5. Error message shown in formatted JSON
6. User compares with "Request" tab to identify issue
7. User copies cURL command to reproduce manually
8. User fixes app code and sees successful request

### Flow 3: Analyzing Performance Over Time

1. User navigates to Dashboard view
2. Latency chart shows spike at 3pm yesterday
3. User clicks on spike time period
4. Activity stream filters to that time range
5. User sees all requests were slow (5s+ duration)
6. User filters to specific provider (OpenAI)
7. User identifies all slow requests used gpt-4 model
8. User shares filtered URL with team

### Flow 4: Searching for Specific Interaction

1. User needs to find request with specific prompt text
2. User enters search term: "analyze this image"
3. Activity stream filters to matching requests
4. User sees 3 results
5. User clicks first result
6. Inspector opens with search term highlighted in request body
7. User copies recording ID for bug report

## Success Metrics

### User Engagement
- % of MIRRA users who access web UI (target: 60% within 3 months)
- Average session duration (target: 5+ minutes)
- Returning users (target: 70% week-over-week)

### Feature Adoption
- % of users using live stream (target: 80%)
- % of users using filters (target: 50%)
- % of users using inspector (target: 90%)
- % of users using dashboard (target: 30%)

### Performance
- Page load time (target: <1s)
- WebSocket connection success rate (target: >95%)
- Time to first recording displayed (target: <100ms)
- Search query response time (target: <500ms for 10k recordings)

### User Satisfaction
- GitHub stars/issues after web UI launch
- User feedback in issues/discussions
- Feature requests related to web UI

## Implementation Phases

### Phase 1: Core MVP (Week 1-2)
**Goal**: Basic viewing and real-time monitoring

- ✓ Embedded web server serving static HTML/CSS/JS
- ✓ Activity stream with live WebSocket updates
- ✓ Basic request inspector (request/response tabs)
- ✓ Simple filtering (provider, date range, status)
- ✓ API endpoints: `/api/recordings`, `/api/recordings/:id`, `/ws/live`
- ✓ Dark theme only
- ✓ Desktop-only layout

**Success Criteria**: User can open UI, see live requests, and inspect details

### Phase 2: Search & Polish (Week 3)
**Goal**: Make it production-ready

- ✓ Full-text search across recordings
- ✓ Advanced filters (model, endpoint, duration, streaming)
- ✓ URL-based filter state (shareable links)
- ✓ Copy buttons (cURL, request, response)
- ✓ Syntax highlighting for JSON
- ✓ Pagination for large result sets
- ✓ Loading states and error handling
- ✓ Light theme option

**Success Criteria**: User can efficiently search and filter 10,000+ recordings

### Phase 3: Analytics (Week 4)
**Goal**: Add insights and observability

- ✓ Dashboard view with stats cards
- ✓ Request volume chart
- ✓ Latency percentiles chart
- ✓ Provider/model/endpoint breakdowns
- ✓ Recent errors list
- ✓ Stats API endpoint with date range filters

**Success Criteria**: User can understand usage patterns at a glance

### Phase 4: Enhancements (Future)
**Goal**: Power user features

- Token counting and cost estimation
- Saved filters/presets
- Request comparison (diff view)
- Export filtered results
- Keyboard shortcuts and command palette
- Settings panel with customization
- Response time alerts/thresholds

## Open Questions & Decisions Needed

1. **WebSocket vs Server-Sent Events**: WebSocket for bidirectional, SSE if we only need server-to-client
   - **Decision**: WebSocket for flexibility (can add client commands later)

2. **Chart Library**: Chart.js, Recharts, custom SVG, or Canvas?
   - **Recommendation**: Chart.js (lightweight, popular) or custom SVG (no dependencies)

3. **Cost Estimation**: Should we include pricing data for token-to-cost calculation?
   - **Recommendation**: Yes, but as optional feature (pricing data in JSON config file)

4. **Token Counting**: Should we parse responses to extract token counts?
   - **Recommendation**: Yes where available (OpenAI returns usage, Claude returns it too)

5. **Request Replay**: In scope for v1 or future?
   - **Recommendation**: Future enhancement (spec mentions it as future)

6. **Multi-instance Support**: What if multiple MIRRA instances run on different ports?
   - **Recommendation**: Each instance has own UI, no cross-instance communication (v1)

7. **Max Recording Size**: Should we limit display of very large request/response bodies?
   - **Recommendation**: Yes, truncate display >1MB with "view full" option

8. **Recording Retention**: Should UI include option to delete old recordings?
   - **Recommendation**: Yes, future enhancement (delete by date range, provider)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Large recordings (GB of JSONL) slow down queries | High | Pagination, indexing (future), date-based file rotation helps |
| WebSocket connection unstable | Medium | Auto-reconnect, fallback to polling, clear error messages |
| Browser compatibility issues | Medium | Target modern browsers only, document requirements |
| Memory leaks from long-running sessions | Medium | Limit buffered recordings, clean up event listeners |
| Sensitive data exposure in UI | High | Reuse existing redaction logic from CLI, add warnings |
| Performance with many concurrent requests | Medium | Virtual scrolling, efficient rendering, Web Workers |

## Future Enhancements (Post-v1)

### Cost & Budget Tracking
- Parse token usage from responses
- Calculate costs based on provider pricing
- Budget alerts and thresholds
- Cost breakdown by team/project (requires tagging)

### Advanced Analytics
- Request patterns and trending prompts
- Model comparison (latency, cost, error rate)
- Anomaly detection (sudden latency spikes)
- Custom metrics and dashboards

### Collaboration Features
- Share recordings with annotations
- Team dashboards (requires auth)
- Comment threads on requests
- Export to external tools (DataDog, etc.)

### Request Manipulation
- Edit and replay recorded requests
- Create request templates
- A/B test different prompts
- Mock responses for testing

### Integrations
- Export to Jupyter notebooks
- Slack/Discord alerts
- CSV/Excel export for business users
- API for programmatic access

## Appendix

### A. Reference Designs

**Inspiration**:
- Chrome DevTools Network Tab: Timeline view, request details, filtering
- Postman: Request/response inspection, syntax highlighting
- DataDog APM: Performance charts, latency analysis
- Sentry: Error grouping, timeline visualization
- Railway.app: Clean dashboard, real-time logs

### B. Technical Constraints

- **No External Dependencies**: Must work in air-gapped environments
- **Single Binary**: Everything embedded in Go binary
- **Backward Compatible**: Must not break existing CLI functionality
- **Low Overhead**: Web UI should not impact proxy performance
- **JSONL Storage**: Must use existing file format, no DB required (v1)

### C. Documentation Requirements

- Update README with web UI section
- Add screenshots/GIFs to documentation
- Create QUICKSTART guide with web UI walkthrough
- Document API endpoints (OpenAPI spec)
- Add troubleshooting section (browser requirements, connection issues)

### D. Testing Strategy

- **Manual Testing**: Click through all flows, test in Chrome/Firefox/Safari
- **Load Testing**: Verify performance with 10k+ recordings
- **WebSocket Testing**: Test reconnection, high-frequency updates
- **Browser Testing**: Test in incognito (no localStorage), slow connections
- **Edge Cases**: Empty state, error states, malformed data

### E. Accessibility Considerations

- Keyboard navigation for all interactive elements
- Focus indicators on all focusable elements
- Alt text for icons (or ARIA labels)
- Sufficient color contrast (WCAG AA)
- Screen reader friendly (semantic HTML)
- Reduced motion option for animations
