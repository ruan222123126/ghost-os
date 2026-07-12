// @vitest-environment jsdom
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { StatusMessage } from "../mobileTypes";

const mocks = vi.hoisted(() => ({
  appVisible: true,
  showNotification: vi.fn(),
  trackerOptions: undefined as {
    onCompleted: (event: CompletionEvent) => Promise<void>;
    onError: (error: unknown) => void;
    visible: boolean;
  } | undefined,
}));

interface CompletionEvent {
  notificationKey: string;
  sessionId: string;
  title: string;
  traceId: string;
}

vi.mock("./useMobileAppVisibility", () => ({
  useMobileAppVisibility: () => mocks.appVisible,
}));

vi.mock("./useMobileNotificationPermission", () => ({
  useMobileNotificationPermission: () => "granted",
}));

vi.mock("./useMobileSessionCompletionTracker", () => ({
  useMobileSessionCompletionTracker: (options: typeof mocks.trackerOptions) => {
    mocks.trackerOptions = options;
  },
}));

vi.mock("../lib/mobileNotifications", () => ({
  clearPendingNotificationSessionId: vi.fn(),
  listenForMobileNotificationOpen: vi.fn(async () => () => undefined),
  loadPendingNotificationSessionId: vi.fn(() => ""),
  showSessionCompletionNotification: mocks.showNotification,
}));

import { useMobileSessionCompletionNotifications } from "./useMobileSessionCompletionNotifications";

describe("useMobileSessionCompletionNotifications", () => {
  beforeEach(() => {
    mocks.appVisible = true;
    mocks.showNotification.mockReset().mockResolvedValue("shown");
    mocks.trackerOptions = undefined;
  });

  it("shows an in-app card when another session completes while visible", async () => {
    const onInAppCompletion = vi.fn();
    renderCompletionHook({ activeSessionId: "session-2", onInAppCompletion });

    await act(async () => {
      await mocks.trackerOptions?.onCompleted(completionEvent());
    });

    expect(onInAppCompletion).toHaveBeenCalledWith(completionEvent());
    expect(mocks.showNotification).not.toHaveBeenCalled();
  });

  it("does not notify for the session currently being watched", async () => {
    const onInAppCompletion = vi.fn();
    renderCompletionHook({ activeSessionId: "session-1", onInAppCompletion });

    await act(async () => {
      await mocks.trackerOptions?.onCompleted(completionEvent());
    });

    expect(onInAppCompletion).not.toHaveBeenCalled();
    expect(mocks.showNotification).not.toHaveBeenCalled();
  });

  it("shows a system notification when the app is hidden even if the session is active", async () => {
    mocks.appVisible = false;
    renderCompletionHook({ activeSessionId: "session-1" });

    await act(async () => {
      await mocks.trackerOptions?.onCompleted(completionEvent());
    });

    expect(mocks.trackerOptions?.visible).toBe(false);
    expect(mocks.showNotification).toHaveBeenCalledWith(completionEvent());
  });

  it("surfaces tracker failures in the application status", () => {
    const onStatus = vi.fn();
    renderCompletionHook({ onStatus });

    act(() => {
      mocks.trackerOptions?.onError(new Error("unsupported action SESSION_RUN_STATES_GET"));
    });

    expect(onStatus).toHaveBeenCalledWith({
      tone: "error",
      text: "会话完成状态跟踪失败：unsupported action SESSION_RUN_STATES_GET",
    });
  });
});

function renderCompletionHook(overrides: {
  activeSessionId?: string;
  onInAppCompletion?: (event: CompletionEvent) => void;
  onStatus?: (status: StatusMessage) => void;
} = {}) {
  return renderHook(() => useMobileSessionCompletionNotifications({
    activeSessionId: overrides.activeSessionId,
    connected: true,
    connectionScope: "bridge-a",
    getSessionRunStates: vi.fn(async () => []),
    knownSessionIds: ["session-1", "session-2"],
    liveSessionRuns: [],
    onInAppCompletion: overrides.onInAppCompletion ?? vi.fn(),
    onStatus: overrides.onStatus ?? vi.fn(),
    selectSession: vi.fn(async () => undefined),
    sessionsLoaded: true,
  }));
}

function completionEvent(): CompletionEvent {
  return {
    notificationKey: "session:session-1:trace:trace-1:status:success",
    sessionId: "session-1",
    title: "Session 1",
    traceId: "trace-1",
  };
}
