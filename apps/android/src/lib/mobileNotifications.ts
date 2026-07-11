import { invoke } from "@tauri-apps/api/core";
import {
  Importance,
  Visibility,
  channels,
  createChannel,
  isPermissionGranted,
  onAction,
  requestPermission,
  sendNotification,
  type Options,
} from "@tauri-apps/plugin-notification";
import type { SessionCompletionEvent } from "./mobileSessionRunTracker";
import { hasTauriRuntime } from "./bridgeBus";

export const SESSION_COMPLETION_NOTIFICATION_CHANNEL_ID = "session_completion";
export const MOBILE_NOTIFICATION_PERMISSION_PROMPTED_KEY = "ghost-os-mobile.notificationPermissionPrompted.v1";

const MOBILE_NOTIFICATION_RETRY_QUEUE_KEY = "ghost-os-mobile.notificationRetryQueue.v1";
const MOBILE_NOTIFICATION_PERMISSION_STATE_KEY = "ghost-os-mobile.notificationPermissionState.v1";
const PENDING_NOTIFICATION_SESSION_KEY = "ghost-os-mobile.pendingNotificationSession.v1";

export type MobileNotificationPermissionState = "unsupported" | "unknown" | "granted" | "denied";
export type MobileNotificationSendResult = "shown" | "duplicate" | "permission_denied" | "unsupported";

let channelReadyPromise: Promise<void> | undefined;

export async function currentMobileNotificationPermission(): Promise<MobileNotificationPermissionState> {
  if (!hasTauriRuntime()) {
    return "unsupported";
  }
  if (await isPermissionGranted()) {
    savePermissionState("granted");
    return "granted";
  }
  return window.localStorage.getItem(MOBILE_NOTIFICATION_PERMISSION_STATE_KEY) === "denied"
    ? "denied"
    : "unknown";
}

export async function requestMobileNotificationPermission(): Promise<MobileNotificationPermissionState> {
  if (!hasTauriRuntime()) {
    return "unsupported";
  }
  if (await isPermissionGranted()) {
    savePermissionState("granted");
    return "granted";
  }
  const permission = await requestPermission();
  const state = permission === "granted" ? "granted" : "denied";
  savePermissionState(state);
  return state;
}

export async function showSessionCompletionNotification(
  event: SessionCompletionEvent,
): Promise<MobileNotificationSendResult> {
  if (!hasTauriRuntime()) {
    return "unsupported";
  }
  if (!await isPermissionGranted()) {
    enqueueNotificationRetry(event);
    return "permission_denied";
  }
  if (await notificationKeyDisplayed(event.notificationKey)) {
    removeNotificationRetry(event.notificationKey);
    return "duplicate";
  }

  try {
    await ensureSessionCompletionNotificationChannel();
    sendNotification({
      autoCancel: true,
      body: `${event.title.trim() || "该会话"}会话已完成`,
      channelId: SESSION_COMPLETION_NOTIFICATION_CHANNEL_ID,
      extra: {
        notification_key: event.notificationKey,
        session_id: event.sessionId,
        trace_id: event.traceId,
      },
      id: notificationId(event.notificationKey),
      title: "会话已完成",
      visibility: Visibility.Private,
    });
    await recordDisplayedNotificationKey(event.notificationKey);
    removeNotificationRetry(event.notificationKey);
    return "shown";
  } catch (error) {
    enqueueNotificationRetry(event);
    throw error;
  }
}

export async function retryPendingMobileNotifications(): Promise<void> {
  for (const event of loadNotificationRetryQueue()) {
    const result = await showSessionCompletionNotification(event);
    if (result === "permission_denied") {
      return;
    }
  }
}

export async function listenForMobileNotificationOpen(
  onOpen: (sessionId: string) => void,
): Promise<() => void> {
  if (!hasTauriRuntime()) {
    return () => undefined;
  }
  const listener = await onAction((notification: Options) => {
    const sessionId = extraString(notification.extra, "session_id");
    if (!sessionId) {
      return;
    }
    savePendingNotificationSessionId(sessionId);
    onOpen(sessionId);
  });
  return () => listener.unregister();
}

export function loadPendingNotificationSessionId(): string {
  return window.localStorage.getItem(PENDING_NOTIFICATION_SESSION_KEY)?.trim() || "";
}

export function savePendingNotificationSessionId(sessionId: string): void {
  const trimmed = sessionId.trim();
  if (!trimmed) {
    return;
  }
  window.localStorage.setItem(PENDING_NOTIFICATION_SESSION_KEY, trimmed);
}

export function clearPendingNotificationSessionId(): void {
  window.localStorage.removeItem(PENDING_NOTIFICATION_SESSION_KEY);
}

async function ensureSessionCompletionNotificationChannel(): Promise<void> {
  channelReadyPromise ??= ensureChannel();
  return channelReadyPromise;
}

async function ensureChannel(): Promise<void> {
  const existing = await channels();
  if (existing.some((channel) => channel.id === SESSION_COMPLETION_NOTIFICATION_CHANNEL_ID)) {
    return;
  }
  await createChannel({
    description: "Ghost-OS 会话完成提醒",
    id: SESSION_COMPLETION_NOTIFICATION_CHANNEL_ID,
    importance: Importance.Default,
    name: "会话完成",
    vibration: true,
    visibility: Visibility.Private,
  });
}

async function notificationKeyDisplayed(notificationKey: string): Promise<boolean> {
  return invoke<boolean>("mobile_notification_key_contains", { notificationKey });
}

async function recordDisplayedNotificationKey(notificationKey: string): Promise<void> {
  await invoke("mobile_notification_key_record", { notificationKey });
}

function notificationId(notificationKey: string): number {
  let hash = 0x811c9dc5;
  for (const character of notificationKey) {
    hash ^= character.charCodeAt(0);
    hash = Math.imul(hash, 0x01000193);
  }
  return hash & 0x7fffffff;
}

function enqueueNotificationRetry(event: SessionCompletionEvent): void {
  const current = loadNotificationRetryQueue();
  const next = [...current.filter((item) => item.notificationKey !== event.notificationKey), event];
  window.localStorage.setItem(MOBILE_NOTIFICATION_RETRY_QUEUE_KEY, JSON.stringify(next));
}

function removeNotificationRetry(notificationKey: string): void {
  const current = loadNotificationRetryQueue();
  const next = current.filter((item) => item.notificationKey !== notificationKey);
  if (next.length === 0) {
    window.localStorage.removeItem(MOBILE_NOTIFICATION_RETRY_QUEUE_KEY);
    return;
  }
  window.localStorage.setItem(MOBILE_NOTIFICATION_RETRY_QUEUE_KEY, JSON.stringify(next));
}

function loadNotificationRetryQueue(): SessionCompletionEvent[] {
  const raw = window.localStorage.getItem(MOBILE_NOTIFICATION_RETRY_QUEUE_KEY);
  if (!raw) {
    return [];
  }
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      throw new Error("notification retry queue must be an array");
    }
    return parsed.filter(isSessionCompletionEvent);
  } catch (error) {
    console.error("[mobileNotifications] load retry queue failed", error);
    return [];
  }
}

function isSessionCompletionEvent(value: unknown): value is SessionCompletionEvent {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return false;
  }
  const record = value as Record<string, unknown>;
  return typeof record.notificationKey === "string"
    && typeof record.sessionId === "string"
    && typeof record.title === "string"
    && typeof record.traceId === "string";
}

function extraString(extra: Record<string, unknown> | undefined, key: string): string {
  const value = extra?.[key];
  return typeof value === "string" ? value.trim() : "";
}

function savePermissionState(state: "granted" | "denied"): void {
  window.localStorage.setItem(MOBILE_NOTIFICATION_PERMISSION_STATE_KEY, state);
}
