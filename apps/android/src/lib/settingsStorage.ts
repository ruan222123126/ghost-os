import type { StoredSettings } from "../mobileTypes";

export const DEFAULT_BRIDGE_URL = "http://127.0.0.1:8080";

const SETTINGS_STORAGE_KEY = "ghost-os-mobile.settings";

const defaultSettings: StoredSettings = {
  bridgeUrl: DEFAULT_BRIDGE_URL,
  sessionId: "",
};

export function loadSettings(): StoredSettings {
  const raw = window.localStorage.getItem(SETTINGS_STORAGE_KEY);
  if (!raw) {
    return defaultSettings;
  }

  try {
    const parsed = JSON.parse(raw) as Partial<StoredSettings>;
    return {
      bridgeUrl: parsed.bridgeUrl?.trim() || DEFAULT_BRIDGE_URL,
      sessionId: parsed.sessionId?.trim() || "",
    };
  } catch {
    return defaultSettings;
  }
}

export function saveSettings(settings: StoredSettings): void {
  window.localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings));
}

export function normalizeBridgeUrl(value: string): string {
  return value.trim().replace(/\/+$/, "");
}
