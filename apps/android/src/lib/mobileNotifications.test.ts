// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  actionCallback: undefined as ((notification: { extra?: Record<string, unknown> }) => void) | undefined,
  channels: vi.fn(),
  createChannel: vi.fn(),
  invoke: vi.fn(),
  isPermissionGranted: vi.fn(),
  onAction: vi.fn(),
  requestPermission: vi.fn(),
  sendNotification: vi.fn(),
  unregister: vi.fn(),
}));

vi.mock("@tauri-apps/api/core", () => ({
  invoke: mocks.invoke,
}));

vi.mock("@tauri-apps/plugin-notification", () => ({
  Importance: { Default: 3 },
  Visibility: { Private: 0 },
  channels: mocks.channels,
  createChannel: mocks.createChannel,
  isPermissionGranted: mocks.isPermissionGranted,
  onAction: mocks.onAction,
  requestPermission: mocks.requestPermission,
  sendNotification: mocks.sendNotification,
}));

vi.mock("./bridgeBus", () => ({
  hasTauriRuntime: () => true,
}));

import {
  listenForMobileNotificationOpen,
  loadPendingNotificationSessionId,
  showSessionCompletionNotification,
} from "./mobileNotifications";

describe("mobileNotifications", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.clearAllMocks();
    mocks.channels.mockResolvedValue([]);
    mocks.createChannel.mockResolvedValue(undefined);
    mocks.isPermissionGranted.mockResolvedValue(true);
    mocks.onAction.mockImplementation(async (callback) => {
      mocks.actionCallback = callback;
      return { unregister: mocks.unregister };
    });
  });

  it("shows a completion notification and records its shared key", async () => {
    mocks.invoke.mockResolvedValueOnce(false).mockResolvedValueOnce(undefined);

    const result = await showSessionCompletionNotification(completionEvent());

    expect(result).toBe("shown");
    expect(mocks.sendNotification).toHaveBeenCalledWith(expect.objectContaining({
      channelId: "session_completion",
      extra: {
        notification_key: "session:session-1:trace:trace-1:status:success",
        session_id: "session-1",
        trace_id: "trace-1",
      },
    }));
    expect(mocks.invoke).toHaveBeenNthCalledWith(2, "mobile_notification_key_record", {
      notificationKey: "session:session-1:trace:trace-1:status:success",
    });
  });

  it("does not display a duplicate notification", async () => {
    mocks.invoke.mockResolvedValueOnce(true);

    expect(await showSessionCompletionNotification(completionEvent())).toBe("duplicate");
    expect(mocks.sendNotification).not.toHaveBeenCalled();
  });

  it("returns an explicit permission denial without recording success", async () => {
    mocks.isPermissionGranted.mockResolvedValue(false);

    expect(await showSessionCompletionNotification(completionEvent())).toBe("permission_denied");
    expect(mocks.invoke).not.toHaveBeenCalled();
    expect(mocks.sendNotification).not.toHaveBeenCalled();
  });

  it("stores and forwards the session selected from a notification action", async () => {
    const onOpen = vi.fn();
    const stop = await listenForMobileNotificationOpen(onOpen);

    mocks.actionCallback?.({ extra: { session_id: "session-1" } });

    expect(onOpen).toHaveBeenCalledWith("session-1");
    expect(loadPendingNotificationSessionId()).toBe("session-1");
    stop();
    expect(mocks.unregister).toHaveBeenCalled();
  });
});

function completionEvent() {
  return {
    notificationKey: "session:session-1:trace:trace-1:status:success",
    sessionId: "session-1",
    title: "Session 1",
    traceId: "trace-1",
  };
}
