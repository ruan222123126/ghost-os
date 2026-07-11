import type { SessionRunState } from "../mobileTypes";
import type { SessionRunStatesGetRequest } from "../mobileTypes";
import {
  loadMobileSessionRunTrackerState,
  saveMobileSessionRunTrackerState,
} from "./mobileSessionRunTrackerStorage";

export const RECENT_SESSION_RECONCILE_LIMIT = 30;
export const FOREGROUND_POLL_INTERVAL_MS = 3_000;
export const BACKGROUND_POLL_INTERVAL_MS = 15_000;
export const POLL_INTERVAL_JITTER_RATIO = 0.1;

export type SessionRunStatus = SessionRunState["status"];

export interface TrackedSessionRun {
  notificationKey?: string;
  sessionId: string;
  status: SessionRunStatus;
  title: string;
  traceId: string;
  updatedAt: string;
}

export interface TrackerState {
  connectionEpoch: number;
  sessions: Record<string, TrackedSessionRun>;
}

export interface SessionCompletionEvent {
  notificationKey: string;
  sessionId: string;
  title: string;
  traceId: string;
}

export interface ReconcileSessionRunsResult {
  completed: SessionCompletionEvent[];
  hotSessionIds: Set<string>;
  sessions: Record<string, TrackedSessionRun>;
}

interface MobileSessionCompletionTrackerRuntimeOptions {
  getSessionRunStates: (params: SessionRunStatesGetRequest) => Promise<SessionRunState[]>;
  onCompleted: (event: SessionCompletionEvent) => void | Promise<void>;
  visible: boolean;
}

export class MobileSessionCompletionTrackerRuntime {
  private connected = false;
  private connectionEpoch = 0;
  private connectionScope = "";
  private getSessionRunStates: MobileSessionCompletionTrackerRuntimeOptions["getSessionRunStates"];
  private hotSessionIds = new Set<string>();
  private inFlight = false;
  private liveRunningSessions: TrackedSessionRun[] = [];
  private onCompleted: MobileSessionCompletionTrackerRuntimeOptions["onCompleted"];
  private pendingReconcileEpoch: number | null = null;
  private sessions: Record<string, TrackedSessionRun> = {};
  private timer: number | null = null;
  private visible: boolean;

  constructor(options: MobileSessionCompletionTrackerRuntimeOptions) {
    this.getSessionRunStates = options.getSessionRunStates;
    this.onCompleted = options.onCompleted;
    this.visible = options.visible;
  }

  updateCallbacks(options: Pick<MobileSessionCompletionTrackerRuntimeOptions, "getSessionRunStates" | "onCompleted">): void {
    this.getSessionRunStates = options.getSessionRunStates;
    this.onCompleted = options.onCompleted;
  }

  updateLiveRunningSessions(running: TrackedSessionRun[]): void {
    this.liveRunningSessions = running;
    this.seedLiveRuns();
    this.scheduleNextPoll();
  }

  setVisible(visible: boolean): void {
    this.visible = visible;
    this.scheduleNextPoll();
  }

  connect(scope: string): void {
    this.disconnect();
    this.connected = true;
    this.connectionScope = scope;
    this.connectionEpoch += 1;
    this.sessions = loadMobileSessionRunTrackerState(scope).sessions;
    this.hotSessionIds = new Set();
    this.seedLiveRuns();
    this.pendingReconcileEpoch = this.connectionEpoch;
    void this.reconcileRecentSessions(this.connectionEpoch);
  }

  disconnect(): void {
    this.connected = false;
    this.connectionEpoch += 1;
    this.pendingReconcileEpoch = null;
    this.clearPollTimer();
  }

  private seedLiveRuns(): void {
    const seeded = seedTrackedRunningSessions(this.sessions, this.liveRunningSessions);
    this.sessions = seeded.sessions;
    this.hotSessionIds = seeded.hotSessionIds;
    this.persist();
  }

  private async reconcileRecentSessions(epoch: number): Promise<void> {
    if (this.inFlight) {
      this.pendingReconcileEpoch = epoch;
      return;
    }
    this.pendingReconcileEpoch = null;
    this.inFlight = true;
    try {
      const states = await this.getSessionRunStates({ limit: RECENT_SESSION_RECONCILE_LIMIT });
      if (this.isCurrentEpoch(epoch)) {
        this.commitReconciliation(states, true);
      }
    } catch (error) {
      if (this.isCurrentEpoch(epoch)) {
        console.error("[MobileSessionCompletionTrackerRuntime] reconcile failed", error);
      }
    } finally {
      this.finishRequest(epoch);
    }
  }

  private async pollTrackedSessions(epoch: number): Promise<void> {
    if (this.inFlight || this.hotSessionIds.size === 0) {
      this.scheduleNextPoll();
      return;
    }
    this.inFlight = true;
    try {
      const states = await this.getSessionRunStates({ session_ids: [...this.hotSessionIds] });
      if (this.isCurrentEpoch(epoch)) {
        this.commitReconciliation(states, false);
      }
    } catch (error) {
      if (this.isCurrentEpoch(epoch)) {
        console.error("[MobileSessionCompletionTrackerRuntime] poll failed", error);
      }
    } finally {
      this.finishRequest(epoch);
    }
  }

  private finishRequest(epoch: number): void {
    this.inFlight = false;
    const pendingEpoch = this.pendingReconcileEpoch;
    if (pendingEpoch !== null && this.isCurrentEpoch(pendingEpoch)) {
      void this.reconcileRecentSessions(pendingEpoch);
      return;
    }
    if (this.isCurrentEpoch(epoch)) {
      this.scheduleNextPoll();
    }
  }

  private commitReconciliation(states: SessionRunState[], replace: boolean): void {
    const result = reconcileSessionRuns(this.sessions, states, replace);
    const seeded = seedTrackedRunningSessions(result.sessions, this.liveRunningSessions);
    this.sessions = seeded.sessions;
    this.hotSessionIds = seeded.hotSessionIds;
    this.persist();
    for (const completion of result.completed) {
      void Promise.resolve(this.onCompleted(completion)).catch((error: unknown) => {
        console.error("[MobileSessionCompletionTrackerRuntime] completion callback failed", error);
      });
    }
  }

  private scheduleNextPoll(): void {
    this.clearPollTimer();
    if (!this.connected || this.hotSessionIds.size === 0) {
      return;
    }
    const epoch = this.connectionEpoch;
    const interval = this.visible ? FOREGROUND_POLL_INTERVAL_MS : BACKGROUND_POLL_INTERVAL_MS;
    this.timer = window.setTimeout(() => {
      this.timer = null;
      void this.pollTrackedSessions(epoch);
    }, jitteredPollInterval(interval));
  }

  private clearPollTimer(): void {
    if (this.timer !== null) {
      window.clearTimeout(this.timer);
      this.timer = null;
    }
  }

  private persist(): void {
    if (!this.connectionScope) {
      return;
    }
    saveMobileSessionRunTrackerState(this.connectionScope, {
      connectionEpoch: this.connectionEpoch,
      sessions: this.sessions,
    });
  }

  private isCurrentEpoch(epoch: number): boolean {
    return this.connected && this.connectionEpoch === epoch;
  }
}

export function reconcileSessionRuns(
  previous: Record<string, TrackedSessionRun>,
  incoming: SessionRunState[],
  replace: boolean,
): ReconcileSessionRunsResult {
  const sessions = replace ? {} : { ...previous };
  const completed: SessionCompletionEvent[] = [];
  const hotSessionIds = new Set<string>();

  for (const raw of incoming) {
    const next = normalizeTrackedSessionRun(raw);
    if (!next) {
      continue;
    }
    const current = previous[next.sessionId];
    if (shouldNotifyCompletion(current, next)) {
      const notificationKey = sessionRunNotificationKey(next);
      next.notificationKey = notificationKey;
      completed.push({
        notificationKey,
        sessionId: next.sessionId,
        title: next.title,
        traceId: next.traceId,
      });
    }
    sessions[next.sessionId] = next;
    if (isIncompleteSessionRunStatus(next.status)) {
      hotSessionIds.add(next.sessionId);
    }
  }

  if (!replace) {
    for (const run of Object.values(sessions)) {
      if (isIncompleteSessionRunStatus(run.status)) {
        hotSessionIds.add(run.sessionId);
      }
    }
  }

  return { completed, hotSessionIds, sessions };
}

export function seedTrackedRunningSessions(
  sessions: Record<string, TrackedSessionRun>,
  running: TrackedSessionRun[],
): ReconcileSessionRunsResult {
  const next = { ...sessions };
  for (const run of running) {
    if (!run.sessionId.trim() || !run.traceId.trim() || !isIncompleteSessionRunStatus(run.status)) {
      continue;
    }
    const current = next[run.sessionId];
    if (current?.traceId === run.traceId && !isIncompleteSessionRunStatus(current.status)) {
      continue;
    }
    next[run.sessionId] = { ...run };
  }
  const hotSessionIds = new Set(
    Object.values(next)
      .filter((run) => isIncompleteSessionRunStatus(run.status))
      .map((run) => run.sessionId),
  );
  return { completed: [], hotSessionIds, sessions: next };
}

export function isIncompleteSessionRunStatus(status: SessionRunStatus): boolean {
  return status === "running" || status === "awaiting_human";
}

export function sessionRunNotificationKey(run: Pick<TrackedSessionRun, "sessionId" | "traceId" | "status">): string {
  return `session:${run.sessionId}:trace:${run.traceId}:status:${run.status}`;
}

export function jitteredPollInterval(baseIntervalMs: number, randomValue = Math.random()): number {
  const boundedRandom = Math.min(1, Math.max(0, randomValue));
  const jitter = (boundedRandom * 2 - 1) * POLL_INTERVAL_JITTER_RATIO;
  return Math.round(baseIntervalMs * (1 + jitter));
}

function normalizeTrackedSessionRun(raw: SessionRunState): TrackedSessionRun | null {
  const sessionId = raw.session_id.trim();
  const traceId = raw.trace_id.trim();
  if (!sessionId) {
    return null;
  }
  if (raw.status !== "idle" && !traceId) {
    return null;
  }
  return {
    sessionId,
    status: raw.status,
    title: raw.title.trim() || sessionId,
    traceId,
    updatedAt: raw.updated_at,
  };
}

function shouldNotifyCompletion(
  previous: TrackedSessionRun | undefined,
  next: TrackedSessionRun,
): boolean {
  return next.status === "success"
    && previous?.traceId === next.traceId
    && isIncompleteSessionRunStatus(previous.status);
}
