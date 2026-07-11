import { useCallback, useEffect, useState } from "react";
import type { SessionRunState, SessionRunStatesGetRequest, StatusMessage } from "../mobileTypes";
import type { SessionCompletionEvent, TrackedSessionRun } from "../lib/mobileSessionRunTracker";
import {
  clearPendingNotificationSessionId,
  listenForMobileNotificationOpen,
  loadPendingNotificationSessionId,
  showSessionCompletionNotification,
} from "../lib/mobileNotifications";
import { useMobileNotificationPermission } from "./useMobileNotificationPermission";
import { useMobileSessionCompletionTracker } from "./useMobileSessionCompletionTracker";

interface UseMobileSessionCompletionNotificationsOptions {
  activeSessionId?: string;
  connected: boolean;
  connectionScope: string;
  getSessionRunStates: (params: SessionRunStatesGetRequest) => Promise<SessionRunState[]>;
  knownSessionIds: string[];
  liveRunningSessions: TrackedSessionRun[];
  onInAppCompletion: (event: SessionCompletionEvent) => void;
  onStatus: (status: StatusMessage) => void;
  selectSession: (sessionId: string) => Promise<void>;
  sessionsLoaded: boolean;
}

export function useMobileSessionCompletionNotifications(
  options: UseMobileSessionCompletionNotificationsOptions,
): void {
  const [notificationOpenToken, setNotificationOpenToken] = useState(0);
  const handlePermissionDenied = useCallback((): void => {
    options.onStatus({ tone: "error", text: "系统通知权限未开启" });
  }, [options.onStatus]);
  useMobileNotificationPermission({
    connected: options.connected,
    onDenied: handlePermissionDenied,
  });

  const handleCompleted = useCallback(async (event: SessionCompletionEvent): Promise<void> => {
    if (event.sessionId === options.activeSessionId) {
      return;
    }
    if (document.visibilityState === "visible") {
      options.onInAppCompletion(event);
      return;
    }
    const result = await showSessionCompletionNotification(event);
    if (result === "permission_denied") {
      options.onStatus({ tone: "error", text: "会话已完成，但系统通知权限未开启" });
    }
  }, [options.activeSessionId, options.onInAppCompletion, options.onStatus]);

  useMobileSessionCompletionTracker({
    connected: options.connected,
    connectionScope: options.connectionScope,
    getSessionRunStates: options.getSessionRunStates,
    liveRunningSessions: options.liveRunningSessions,
    onCompleted: handleCompleted,
  });

  useEffect(() => {
    let cancelled = false;
    let stopListening: (() => void) | undefined;
    void listenForMobileNotificationOpen(() => {
      setNotificationOpenToken((current) => current + 1);
    })
      .then((stop) => {
        if (cancelled) {
          stop();
          return;
        }
        stopListening = stop;
      })
      .catch((error: unknown) => {
        console.error("[useMobileSessionCompletionNotifications] action listener failed", error);
      });
    return () => {
      cancelled = true;
      stopListening?.();
    };
  }, []);

  useEffect(() => {
    if (!options.connected || !options.sessionsLoaded) {
      return;
    }
    const sessionId = loadPendingNotificationSessionId();
    if (!sessionId) {
      return;
    }
    if (!options.knownSessionIds.includes(sessionId)) {
      clearPendingNotificationSessionId();
      options.onStatus({ tone: "error", text: `通知对应的会话不存在：${sessionId}` });
      return;
    }

    clearPendingNotificationSessionId();
    void options.selectSession(sessionId).catch((error: unknown) => {
      options.onStatus({ tone: "error", text: `打开通知会话失败：${String(error)}` });
    });
  }, [
    notificationOpenToken,
    options.connected,
    options.knownSessionIds,
    options.onStatus,
    options.selectSession,
    options.sessionsLoaded,
  ]);
}
