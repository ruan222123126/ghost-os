import { useEffect, useRef } from "react";
import type { SessionRunState, SessionRunStatesGetRequest } from "../mobileTypes";
import {
  MobileSessionCompletionTrackerRuntime,
  type SessionCompletionEvent,
  type TrackedSessionRun,
} from "../lib/mobileSessionRunTracker";

interface UseMobileSessionCompletionTrackerOptions {
  connected: boolean;
  connectionScope: string;
  getSessionRunStates: (params: SessionRunStatesGetRequest) => Promise<SessionRunState[]>;
  liveRunningSessions: TrackedSessionRun[];
  onCompleted: (event: SessionCompletionEvent) => void | Promise<void>;
}

export function useMobileSessionCompletionTracker(
  options: UseMobileSessionCompletionTrackerOptions,
): void {
  const runtimeRef = useRef<MobileSessionCompletionTrackerRuntime | null>(null);
  runtimeRef.current ??= new MobileSessionCompletionTrackerRuntime({
    getSessionRunStates: options.getSessionRunStates,
    onCompleted: options.onCompleted,
    visible: isAppVisible(),
  });
  const runtime = runtimeRef.current;

  useEffect(() => {
    runtime.updateCallbacks({
      getSessionRunStates: options.getSessionRunStates,
      onCompleted: options.onCompleted,
    });
  }, [options.getSessionRunStates, options.onCompleted, runtime]);

  useEffect(() => {
    runtime.updateLiveRunningSessions(options.liveRunningSessions);
  }, [options.liveRunningSessions, runtime]);

  useEffect(() => {
    const handleVisibilityChange = (): void => runtime.setVisible(isAppVisible());
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => document.removeEventListener("visibilitychange", handleVisibilityChange);
  }, [runtime]);

  useEffect(() => {
    if (options.connected) {
      runtime.connect(options.connectionScope);
    } else {
      runtime.disconnect();
    }
    return () => runtime.disconnect();
  }, [options.connected, options.connectionScope, runtime]);
}

function isAppVisible(): boolean {
  return typeof document === "undefined" || document.visibilityState === "visible";
}
