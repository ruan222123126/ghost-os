// @vitest-environment jsdom
import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { SessionRunState } from "../mobileTypes";
import { useMobileSessionCompletionTracker } from "./useMobileSessionCompletionTracker";

describe("useMobileSessionCompletionTracker", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("reconciles once, then polls only the unfinished session", async () => {
    const getSessionRunStates = vi.fn()
      .mockResolvedValueOnce([runState("running")])
      .mockResolvedValueOnce([runState("success")]);
    const onCompleted = vi.fn();

    renderHook(() => useMobileSessionCompletionTracker({
      connected: true,
      connectionScope: "bridge-a",
      getSessionRunStates,
      liveRunningSessions: [],
      onCompleted,
    }));
    await flushPromises();

    expect(getSessionRunStates).toHaveBeenNthCalledWith(1, { limit: 30 });

    await act(async () => {
      vi.advanceTimersByTime(3_301);
      await Promise.resolve();
    });
    await flushPromises();

    expect(getSessionRunStates).toHaveBeenNthCalledWith(2, { session_ids: ["session-1"] });
    expect(onCompleted).toHaveBeenCalledTimes(1);

    act(() => vi.advanceTimersByTime(20_000));
    expect(getSessionRunStates).toHaveBeenCalledTimes(2);
  });

  it("does not overlap an in-flight poll", async () => {
    let resolvePoll: ((value: SessionRunState[]) => void) | undefined;
    const getSessionRunStates = vi.fn()
      .mockResolvedValueOnce([runState("running")])
      .mockImplementationOnce(() => new Promise<SessionRunState[]>((resolve) => {
        resolvePoll = resolve;
      }));

    renderHook(() => useMobileSessionCompletionTracker({
      connected: true,
      connectionScope: "bridge-a",
      getSessionRunStates,
      liveRunningSessions: [],
      onCompleted: vi.fn(),
    }));
    await flushPromises();

    act(() => vi.advanceTimersByTime(3_301));
    await flushPromises();
    act(() => vi.advanceTimersByTime(20_000));
    expect(getSessionRunStates).toHaveBeenCalledTimes(2);

    await act(async () => {
      resolvePoll?.([runState("success")]);
      await Promise.resolve();
    });
  });

  it("runs the newest connection calibration after an old response finishes", async () => {
    let resolveOld: ((value: SessionRunState[]) => void) | undefined;
    const getSessionRunStates = vi.fn()
      .mockImplementationOnce(() => new Promise<SessionRunState[]>((resolve) => {
        resolveOld = resolve;
      }))
      .mockResolvedValueOnce([runState("success")]);
    const onCompleted = vi.fn();
    const { rerender } = renderHook(
      (props: { connected: boolean; scope: string }) => useMobileSessionCompletionTracker({
        connected: props.connected,
        connectionScope: props.scope,
        getSessionRunStates,
        liveRunningSessions: [],
        onCompleted,
      }),
      { initialProps: { connected: true, scope: "bridge-a" } },
    );

    rerender({ connected: false, scope: "bridge-a" });
    rerender({ connected: true, scope: "bridge-b" });
    expect(getSessionRunStates).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolveOld?.([runState("running")]);
      await Promise.resolve();
      await Promise.resolve();
    });
    await flushPromises();

    expect(getSessionRunStates).toHaveBeenCalledTimes(2);
    expect(getSessionRunStates).toHaveBeenNthCalledWith(2, { limit: 30 });
    expect(onCompleted).not.toHaveBeenCalled();
  });
});

async function flushPromises(): Promise<void> {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}

function runState(status: SessionRunState["status"]): SessionRunState {
  return {
    session_id: "session-1",
    started_at: "2026-07-11T08:00:00Z",
    status,
    terminal_at: status === "success" ? "2026-07-11T08:01:00Z" : "",
    title: "Session 1",
    trace_id: "trace-1",
    updated_at: "2026-07-11T08:01:00Z",
  };
}
