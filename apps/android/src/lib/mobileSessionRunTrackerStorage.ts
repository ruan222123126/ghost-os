import type { TrackerState, TrackedSessionRun } from "./mobileSessionRunTracker";

const MOBILE_SESSION_RUN_TRACKER_STORAGE_PREFIX = "ghost-os-mobile.sessionRunTracker.v1";

export function loadMobileSessionRunTrackerState(scope: string): TrackerState {
  const raw = window.localStorage.getItem(storageKey(scope));
  if (!raw) {
    return emptyTrackerState();
  }
  try {
    return normalizeTrackerState(JSON.parse(raw));
  } catch (error) {
    console.error("[mobileSessionRunTrackerStorage] load failed", error);
    return emptyTrackerState();
  }
}

export function saveMobileSessionRunTrackerState(scope: string, state: TrackerState): void {
  window.localStorage.setItem(storageKey(scope), JSON.stringify(state));
}

function storageKey(scope: string): string {
  return `${MOBILE_SESSION_RUN_TRACKER_STORAGE_PREFIX}:${scope.trim() || "default"}`;
}

function emptyTrackerState(): TrackerState {
  return { connectionEpoch: 0, sessions: {} };
}

function normalizeTrackerState(value: unknown): TrackerState {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error("tracker state must be an object");
  }
  const record = value as Record<string, unknown>;
  const connectionEpoch = Number.isInteger(record.connectionEpoch) && Number(record.connectionEpoch) >= 0
    ? Number(record.connectionEpoch)
    : 0;
  return {
    connectionEpoch,
    sessions: normalizeSessions(record.sessions),
  };
}

function normalizeSessions(value: unknown): Record<string, TrackedSessionRun> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  const sessions: Record<string, TrackedSessionRun> = {};
  for (const [sessionId, raw] of Object.entries(value)) {
    const run = normalizeRun(raw);
    if (run && run.sessionId === sessionId) {
      sessions[sessionId] = run;
    }
  }
  return sessions;
}

function normalizeRun(value: unknown): TrackedSessionRun | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  const record = value as Record<string, unknown>;
  const sessionId = stringValue(record.sessionId);
  const traceId = stringValue(record.traceId);
  const title = stringValue(record.title) || sessionId;
  const updatedAt = stringValue(record.updatedAt);
  const status = record.status;
  if (!sessionId || !isSessionRunStatus(status)) {
    return null;
  }
  return {
    notificationKey: stringValue(record.notificationKey) || undefined,
    sessionId,
    status,
    title,
    traceId,
    updatedAt,
  };
}

function stringValue(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function isSessionRunStatus(value: unknown): value is TrackedSessionRun["status"] {
  return value === "running"
    || value === "awaiting_human"
    || value === "success"
    || value === "error"
    || value === "cancelled"
    || value === "idle";
}
