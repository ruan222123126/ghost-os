import { describe, expect, it } from "vitest";
import type { SessionRunState } from "../mobileTypes";
import {
  FOREGROUND_POLL_INTERVAL_MS,
  jitteredPollInterval,
  reconcileSessionRuns,
  reconcileTrackedSessionRuns,
  seedTrackedRunningSessions,
  sessionRunNotificationKey,
  type TrackedSessionRun,
} from "./mobileSessionRunTracker";

describe("mobileSessionRunTracker", () => {
  it("establishes a completed baseline without notifying", () => {
    const result = reconcileSessionRuns({}, [runState("success")], true);

    expect(result.completed).toEqual([]);
    expect(result.hotSessionIds.size).toBe(0);
    expect(result.sessions["session-1"]?.status).toBe("success");
  });

  it("notifies once when the same trace moves from running to success", () => {
    const previous: Record<string, TrackedSessionRun> = {
      "session-1": trackedRun("running"),
    };

    const first = reconcileSessionRuns(previous, [runState("success")], false);
    const second = reconcileSessionRuns(first.sessions, [runState("success")], false);

    expect(first.completed).toEqual([{
      notificationKey: "session:session-1:trace:trace-1:status:success",
      sessionId: "session-1",
      title: "Session 1",
      traceId: "trace-1",
    }]);
    expect(first.hotSessionIds.size).toBe(0);
    expect(second.completed).toEqual([]);
  });

  it("notifies immediately when a live stream moves from running to success", () => {
    const previous = { "session-1": trackedRun("running") };
    const completedRun = {
      ...trackedRun("success"),
      updatedAt: "2026-07-11T08:01:00Z",
    };

    const result = reconcileTrackedSessionRuns(previous, [completedRun]);

    expect(result.completed).toEqual([{
      notificationKey: "session:session-1:trace:trace-1:status:success",
      sessionId: "session-1",
      title: "Session 1",
      traceId: "trace-1",
    }]);
    expect(result.hotSessionIds.size).toBe(0);
  });

  it("does not treat a new trace terminal state as the previous run completion", () => {
    const previous = { "session-1": trackedRun("running") };
    const next = runState("success", "trace-2");

    expect(reconcileSessionRuns(previous, [next], false).completed).toEqual([]);
  });

  it("keeps only incomplete runs in the hot set", () => {
    const result = reconcileSessionRuns({}, [
      runState("running"),
      { ...runState("awaiting_human"), session_id: "session-2", trace_id: "trace-2" },
      { ...runState("error"), session_id: "session-3", trace_id: "trace-3" },
    ], true);

    expect([...result.hotSessionIds].sort()).toEqual(["session-1", "session-2"]);
  });

  it("does not let a stale local running seed overwrite a terminal server state", () => {
    const terminal = trackedRun("success");
    const seeded = seedTrackedRunningSessions({ "session-1": terminal }, [trackedRun("running")]);

    expect(seeded.sessions["session-1"]?.status).toBe("success");
    expect(seeded.hotSessionIds.size).toBe(0);
  });

  it("applies bounded ten percent jitter", () => {
    expect(jitteredPollInterval(FOREGROUND_POLL_INTERVAL_MS, 0)).toBe(2_700);
    expect(jitteredPollInterval(FOREGROUND_POLL_INTERVAL_MS, 0.5)).toBe(3_000);
    expect(jitteredPollInterval(FOREGROUND_POLL_INTERVAL_MS, 1)).toBe(3_300);
  });

  it("builds the shared notification key", () => {
    expect(sessionRunNotificationKey(trackedRun("success")))
      .toBe("session:session-1:trace:trace-1:status:success");
  });
});

function runState(status: SessionRunState["status"], traceId = "trace-1"): SessionRunState {
  return {
    session_id: "session-1",
    started_at: "2026-07-11T08:00:00Z",
    status,
    terminal_at: status === "success" ? "2026-07-11T08:01:00Z" : "",
    title: "Session 1",
    trace_id: traceId,
    updated_at: "2026-07-11T08:01:00Z",
  };
}

function trackedRun(status: TrackedSessionRun["status"]): TrackedSessionRun {
  return {
    sessionId: "session-1",
    status,
    title: "Session 1",
    traceId: "trace-1",
    updatedAt: "2026-07-11T08:00:00Z",
  };
}
