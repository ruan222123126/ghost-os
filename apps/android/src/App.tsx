import { invoke } from "@tauri-apps/api/core";
import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import "./App.css";

const DEFAULT_BRIDGE_URL = "http://127.0.0.1:8080";
const SETTINGS_STORAGE_KEY = "ghost-os-mobile.settings";

interface HostProfile {
  productName: string;
  version: string;
  target: string;
  mobile: boolean;
}

interface StoredSettings {
  bridgeUrl: string;
  sessionId: string;
}

interface BridgeBusCommand {
  baseUrl: string;
  apiToken?: string;
  action: string;
  params: Record<string, unknown>;
  traceId: string;
}

interface BridgeEnvelope<TPayload> {
  status: "success" | "error";
  payload: TPayload;
  error: string;
}

interface ConfigPayload {
  provider?: string;
  provider_type?: string;
  model?: string;
  project_root?: string;
  api_key_set?: boolean;
}

interface AgentPayload {
  message: string;
  session_id: string;
  session_ended: boolean;
  mode?: string;
}

interface StatusMessage {
  tone: "idle" | "loading" | "success" | "error";
  text: string;
}

const defaultSettings: StoredSettings = {
  bridgeUrl: DEFAULT_BRIDGE_URL,
  sessionId: "",
};

function loadSettings(): StoredSettings {
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

function saveSettings(settings: StoredSettings): void {
  window.localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings));
}

function createTraceId(prefix: string): string {
  const normalizedPrefix = prefix.trim() || "mobile";
  const random = crypto.randomUUID?.() ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
  return `${normalizedPrefix}-${random}`;
}

function normalizeBridgeUrl(value: string): string {
  return value.trim().replace(/\/+$/, "");
}

function isNonEmptyMessage(value: string): boolean {
  return value.trim().length > 0;
}

function App() {
  const [settings, setSettings] = useState<StoredSettings>(() => loadSettings());
  const [apiToken, setApiToken] = useState("");
  const [message, setMessage] = useState("");
  const [host, setHost] = useState<HostProfile>();
  const [config, setConfig] = useState<ConfigPayload>();
  const [reply, setReply] = useState<AgentPayload>();
  const [lastTraceId, setLastTraceId] = useState("");
  const [status, setStatus] = useState<StatusMessage>({
    tone: "idle",
    text: "未连接",
  });

  const bridgeUrl = useMemo(() => normalizeBridgeUrl(settings.bridgeUrl), [settings.bridgeUrl]);
  const canSend = isNonEmptyMessage(message) && status.tone !== "loading";

  useEffect(() => {
    invoke<HostProfile>("host_profile")
      .then(setHost)
      .catch((error: unknown) => {
        setStatus({
          tone: "error",
          text: error instanceof Error ? error.message : String(error),
        });
      });
  }, []);

  useEffect(() => {
    saveSettings(settings);
  }, [settings]);

  async function requestBridge<TPayload>(
    action: string,
    params: Record<string, unknown>,
  ): Promise<TPayload> {
    const traceId = createTraceId(`android-${action.toLowerCase()}`);
    setLastTraceId(traceId);

    const request: BridgeBusCommand = {
      baseUrl: bridgeUrl,
      apiToken: apiToken.trim() || undefined,
      action,
      params,
      traceId,
    };
    const envelope = await invoke<BridgeEnvelope<TPayload>>("bridge_bus_request", { request });
    if (envelope.status === "error") {
      throw new Error(envelope.error || "Bridge returned an error envelope");
    }
    return envelope.payload;
  }

  async function connectBridge(): Promise<void> {
    setStatus({ tone: "loading", text: "连接中" });
    try {
      const payload = await requestBridge<ConfigPayload>("CONFIG_GET", {});
      setConfig(payload);
      setStatus({ tone: "success", text: "Bridge 已连接" });
    } catch (error) {
      setConfig(undefined);
      setStatus({
        tone: "error",
        text: error instanceof Error ? error.message : String(error),
      });
    }
  }

  async function sendMessage(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const trimmed = message.trim();
    if (!trimmed) {
      return;
    }

    setStatus({ tone: "loading", text: "发送中" });
    try {
      const payload = await requestBridge<AgentPayload>("AGENT_SEND", {
        message: trimmed,
        ...(settings.sessionId.trim() ? { session_id: settings.sessionId.trim() } : {}),
      });
      setReply(payload);
      setSettings((current) => ({
        ...current,
        sessionId: payload.session_id || current.sessionId,
      }));
      setMessage("");
      setStatus({ tone: "success", text: "回复已返回" });
    } catch (error) {
      setStatus({
        tone: "error",
        text: error instanceof Error ? error.message : String(error),
      });
    }
  }

  return (
    <main className="app-shell">
      <header className="top-bar">
        <div>
          <p className="eyebrow">Ghost-OS</p>
          <h1>Mobile Console</h1>
        </div>
        <span className={`status-pill status-${status.tone}`}>{status.text}</span>
      </header>

      <section className="panel connection-panel" aria-label="Bridge connection">
        <label className="field">
          <span>Bridge URL</span>
          <input
            value={settings.bridgeUrl}
            inputMode="url"
            spellCheck={false}
            onChange={(event) =>
              setSettings((current) => ({
                ...current,
                bridgeUrl: event.currentTarget.value,
              }))
            }
          />
        </label>
        <label className="field">
          <span>API Token</span>
          <input
            value={apiToken}
            type="password"
            autoComplete="off"
            onChange={(event) => setApiToken(event.currentTarget.value)}
          />
        </label>
        <label className="field">
          <span>Session ID</span>
          <input
            value={settings.sessionId}
            spellCheck={false}
            onChange={(event) =>
              setSettings((current) => ({
                ...current,
                sessionId: event.currentTarget.value,
              }))
            }
          />
        </label>
        <button className="primary-button" type="button" onClick={connectBridge} disabled={status.tone === "loading"}>
          连接
        </button>
      </section>

      <section className="runtime-grid" aria-label="Runtime state">
        <div className="metric">
          <span>Host</span>
          <strong>{host ? `${host.target}${host.mobile ? " mobile" : ""}` : "Tauri"}</strong>
        </div>
        <div className="metric">
          <span>Provider</span>
          <strong>{config?.provider || "-"}</strong>
        </div>
        <div className="metric">
          <span>Model</span>
          <strong>{config?.model || "-"}</strong>
        </div>
        <div className="metric">
          <span>Trace</span>
          <strong>{lastTraceId || "-"}</strong>
        </div>
      </section>

      <form className="panel composer-panel" onSubmit={sendMessage}>
        <label className="field message-field">
          <span>Message</span>
          <textarea
            value={message}
            rows={5}
            onChange={(event) => setMessage(event.currentTarget.value)}
          />
        </label>
        <button className="primary-button" type="submit" disabled={!canSend}>
          发送
        </button>
      </form>

      <section className="response-panel" aria-live="polite">
        <div className="response-header">
          <span>Assistant</span>
          <span>{reply?.session_id || settings.sessionId || "-"}</span>
        </div>
        <p>{reply?.message || "等待 Bridge 响应"}</p>
      </section>
    </main>
  );
}

export default App;
