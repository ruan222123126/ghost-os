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
  liveSessionRuns: TrackedSessionRun[];
  onCompleted: (event: SessionCompletionEvent) => void | Promise<void>;
  onError: (error: unknown) => void;
  visible: boolean;
}

export function useMobileSessionCompletionTracker(
  options: UseMobileSessionCompletionTrackerOptions,
): void {
  const runtimeRef = useRef<MobileSessionCompletionTrackerRuntime | null>(null);
  runtimeRef.current ??= new MobileSessionCompletionTrackerRuntime({
    getSessionRunStates: options.getSessionRunStates,
    onCompleted: options.onCompleted,
    onError: options.onError,
    visible: options.visible,
  });
  const runtime = runtimeRef.current;

  useEffect(() => {
    runtime.updateCallbacks({
      getSessionRunStates: options.getSessionRunStates,
      onCompleted: options.onCompleted,
      onError: options.onError,
    });
  }, [options.getSessionRunStates, options.onCompleted, options.onError, runtime]);

  useEffect(() => {
    runtime.updateLiveSessionRuns(options.liveSessionRuns);
  }, [options.liveSessionRuns, runtime]);

  useEffect(() => {
    runtime.setVisible(options.visible);
  }, [options.visible, runtime]);

  useEffect(() => {
    if (options.connected) {
      runtime.connect(options.connectionScope);
    } else {
      runtime.disconnect();
    }
    return () => runtime.disconnect();
  }, [options.connected, options.connectionScope, runtime]);
}
