import type { StoredSettings } from "../mobileTypes";

export const DEFAULT_BRIDGE_URL = "http://127.0.0.1:8080";

const SETTINGS_STORAGE_KEY = "ghost-os-mobile.settings";

const defaultSettings: StoredSettings = {
  bridgeUrl: DEFAULT_BRIDGE_URL,
  connectionMode: "webrtc",
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
      connectionMode: parsed.connectionMode === "http" ? "http" : "webrtc",
      pairing: normalizePairing(parsed.pairing),
    };
  } catch {
    return defaultSettings;
  }
}

export function saveSettings(settings: StoredSettings): void {
  window.localStorage.setItem(
    SETTINGS_STORAGE_KEY,
    JSON.stringify({
      bridgeUrl: settings.bridgeUrl,
      connectionMode: settings.connectionMode,
      pairing: settings.pairing,
    }),
  );
}

export function normalizeBridgeUrl(value: string): string {
  return value.trim().replace(/\/+$/, "");
}

function normalizePairing(raw: StoredSettings["pairing"]): StoredSettings["pairing"] {
  if (!raw) {
    return undefined;
  }
  const deviceId = raw.deviceId?.trim();
  const pcId = raw.pcId?.trim();
  const signalingUrl = raw.signalingUrl?.trim();
  const signalingToken = raw.signalingToken?.trim();
  if (!deviceId || !pcId || !signalingUrl || !signalingToken) {
    return undefined;
  }
  return {
    deviceId,
    pcId,
    signalingUrl,
    signalingToken,
    iceServers: normalizeIceServers(raw.iceServers),
  };
}

function normalizeIceServers(raw: RTCIceServer[] | undefined): RTCIceServer[] {
  if (!Array.isArray(raw)) {
    return [];
  }
  return raw
    .map((server) => ({
      credential: typeof server.credential === "string" ? server.credential.trim() : server.credential,
      urls: Array.isArray(server.urls)
        ? server.urls.map((url) => url.trim()).filter(Boolean)
        : typeof server.urls === "string"
          ? server.urls.trim()
          : "",
      username: server.username?.trim(),
    }))
    .filter((server) => (Array.isArray(server.urls) ? server.urls.length > 0 : Boolean(server.urls)));
}
