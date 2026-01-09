export interface RecordingSummary {
  id: string;
  timestamp: string;
  provider: string;
  method: string;
  path: string;
  status: number;
  duration: number;
  responseSize: number;
  error?: string;
}

export interface RecordingListResponse {
  recordings: RecordingSummary[];
  total: number;
  page: number;
  limit: number;
  hasMore: boolean;
}

export interface Recording {
  id: string;
  timestamp: string;
  provider: string;
  request: {
    method: string;
    path: string;
    query: string;
    headers: Record<string, string[]>;
    body: any;
  };
  response: {
    status: number;
    headers: Record<string, string[]>;
    body: any;
    streaming: boolean;
  };
  timing: {
    startedAt: string;
    completedAt: string;
    duration_ms: number;
  };
  error?: string;
}

export interface ParsedStream {
  text: string;
  metadata: Record<string, any>;
  eventCounts: Record<string, number>;
}

export async function fetchRecordings(
  page: number,
  limit: number,
  provider?: string,
  search?: string,
): Promise<RecordingListResponse> {
  const params = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  if (provider) params.append("provider", provider);
  if (search) params.append("search", search);

  const response = await fetch(`/api/recordings?${params}`);
  if (!response.ok) {
    throw new Error("Failed to fetch recordings");
  }
  return response.json();
}

/**
 * Fetches a single recording by ID
 */
export async function fetchRecording(id: string): Promise<Recording> {
  const response = await fetch(`/api/recordings/${id}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch recording: ${response.statusText}`);
  }
  return response.json();
}

/**
 * Fetches parsed stream data for a recording
 */
export async function fetchParsedRecording(id: string): Promise<ParsedStream> {
  const response = await fetch(`/api/recordings/${id}/parse`);
  if (!response.ok) {
    throw new Error(`Failed to parse recording: ${response.statusText}`);
  }
  return response.json();
}

// Session Grouping Interfaces and Functions

export interface SessionGroup {
  trace_id: string;
  session_id: string;
  recording_ids: string[];
  first_timestamp: string;
  last_timestamp: string;
  request_count: number;
  providers: string[];
  has_errors: boolean;
}

export interface SessionGroupListResponse {
  groups: SessionGroup[];
  total: number;
  page: number;
  limit: number;
  hasMore: boolean;
}

export interface SessionGroupDetail {
  group: SessionGroup;
  recordings: RecordingSummary[];
}

/**
 * Fetches session groups with optional filters
 */
export async function fetchSessionGroups(
  page: number,
  limit: number,
  provider?: string,
  hasErrors?: boolean,
  fromDate?: string,
  toDate?: string,
): Promise<SessionGroupListResponse> {
  const params = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  if (provider) params.append("provider", provider);
  if (hasErrors !== undefined) params.append("has_errors", hasErrors.toString());
  if (fromDate) params.append("from", fromDate);
  if (toDate) params.append("to", toDate);

  const response = await fetch(`/api/groups/sessions?${params}`);
  if (!response.ok) {
    throw new Error("Failed to fetch session groups");
  }
  return response.json();
}

/**
 * Fetches a single session group with full recording details
 */
export async function fetchSessionGroup(traceId: string): Promise<SessionGroupDetail> {
  const response = await fetch(`/api/groups/sessions/${traceId}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch session group: ${response.statusText}`);
  }
  return response.json();
}
