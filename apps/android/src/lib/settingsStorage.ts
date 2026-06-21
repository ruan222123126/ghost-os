import type { ConnectionMode, StoredConnectionSnapshot, StoredSettings } from "../mobileTypes";

export const DEFAULT_BRIDGE_URL = "http://127.0.0.1:8080";

const SETTINGS_STORAGE_KEY = "ghost-os-mobile.settings";

const defaultSettings: StoredSettings = {
  apiToken: "",
  autoConnectEnabled: false,
  bridgeUrl: DEFAULT_BRIDGE_URL,
  connectionMode: "webrtc",
  persistComputerSessionsEnabled: false,
};

export function loadSettings(): StoredSettings {
  const raw = window.localStorage.getItem(SETTINGS_STORAGE_KEY);
  if (!raw) {
    return { ...defaultSettings };
  }

  try {
    const parsed = JSON.parse(raw) as Partial<StoredSettings>;
    const connectionMode = normalizeConnectionMode(parsed.connectionMode);
    return {
      apiToken: parsed.apiToken?.trim() || "",
      autoConnectEnabled: parsed.autoConnectEnabled === true,
      bridgeUrl: parsed.bridgeUrl?.trim() || DEFAULT_BRIDGE_URL,
      connectionMode,
      lastSuccessfulConnection: normalizeLastSuccessfulConnection(parsed.lastSuccessfulConnection),
      pairing: normalizePairing(parsed.pairing),
      persistComputerSessionsEnabled: parsed.persistComputerSessionsEnabled === true,
    };
  } catch {
    return { ...defaultSettings };
  }
}

export function saveSettings(settings: StoredSettings): void {
  window.localStorage.setItem(
    SETTINGS_STORAGE_KEY,
    JSON.stringify({
      apiToken: settings.apiToken?.trim() || "",
      autoConnectEnabled: settings.autoConnectEnabled,
      bridgeUrl: settings.bridgeUrl,
      connectionMode: settings.connectionMode,
      lastSuccessfulConnection: normalizeLastSuccessfulConnection(settings.lastSuccessfulConnection),
      pairing: settings.pairing,
      persistComputerSessionsEnabled: settings.persistComputerSessionsEnabled,
    }),
  );
}

export function normalizeBridgeUrl(value: string): string {
  return value.trim().replace(/\/+$/, "");
}

function normalizeConnectionMode(value: ConnectionMode | undefined): ConnectionMode {
  return value === "http" ? "http" : "webrtc";
}

function normalizeLastSuccessfulConnection(
  raw: StoredConnectionSnapshot | undefined,
): StoredConnectionSnapshot | undefined {
  if (!raw) {
    return undefined;
  }

  const connectionMode = normalizeConnectionMode(raw.connectionMode);
  const bridgeUrl = raw.bridgeUrl?.trim() || DEFAULT_BRIDGE_URL;
  const apiToken = raw.apiToken?.trim() || "";
  if (connectionMode === "http") {
    return {
      apiToken,
      bridgeUrl,
      connectionMode,
    };
  }

  const pairing = normalizePairing(raw.pairing);
  if (!pairing) {
    return undefined;
  }
  return {
    apiToken,
    bridgeUrl,
    connectionMode,
    pairing,
  };
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
