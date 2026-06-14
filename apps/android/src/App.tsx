import { invoke } from "@tauri-apps/api/core";
import { useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import {
  AssistantIntro,
  AssistantReply,
  BridgeSettings,
  ChatBubble,
  ChatComposer,
  ChatHeader,
} from "./components/MobileChatHome";
import type { AgentPayload, ConfigPayload, HostProfile, StatusMessage, StoredSettings } from "./mobileTypes";
import "./App.css";

const DEFAULT_BRIDGE_URL = "http://127.0.0.1:8080";
const SETTINGS_STORAGE_KEY = "ghost-os-mobile.settings";
const SCROLL_DOWN_THRESHOLD_PX = 50;

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

function shouldShowScrollDown(element: HTMLElement): boolean {
  return element.scrollHeight - element.scrollTop - element.clientHeight > SCROLL_DOWN_THRESHOLD_PX;
}

function displayRuntime(config: ConfigPayload | undefined): string {
  if (config?.provider && config.model) {
    return `${config.provider} / ${config.model}`;
  }
  return config?.provider || config?.model || "Bridge Runtime";
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function App() {
  const [settings, setSettings] = useState<StoredSettings>(() => loadSettings());
  const [apiToken, setApiToken] = useState("");
  const [message, setMessage] = useState("");
  const [host, setHost] = useState<HostProfile>();
  const [config, setConfig] = useState<ConfigPayload>();
  const [reply, setReply] = useState<AgentPayload>();
  const [lastUserMessage, setLastUserMessage] = useState("");
  const [lastTraceId, setLastTraceId] = useState("");
  const [showScrollDown, setShowScrollDown] = useState(false);
  const [status, setStatus] = useState<StatusMessage>({
    tone: "idle",
    text: "未连接",
  });

  const scrollRef = useRef<HTMLElement>(null);
  const settingsRef = useRef<HTMLElement>(null);
  const bridgeUrl = useMemo(() => normalizeBridgeUrl(settings.bridgeUrl), [settings.bridgeUrl]);
  const canSend = isNonEmptyMessage(message) && status.tone !== "loading";
  const runtimeLabel = useMemo(() => displayRuntime(config), [config]);

  useEffect(() => {
    try {
      void invoke<HostProfile>("host_profile")
        .then(setHost)
        .catch((error: unknown) => {
          setStatus({ tone: "error", text: errorMessage(error) });
        });
    } catch (error) {
      setStatus({ tone: "error", text: errorMessage(error) });
    }
  }, []);

  useEffect(() => {
    saveSettings(settings);
  }, [settings]);

  useEffect(() => {
    if (lastUserMessage || reply) {
      scrollToBottom("smooth");
    }
  }, [lastUserMessage, reply]);

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
      setStatus({ tone: "error", text: errorMessage(error) });
    }
  }

  async function sendMessage(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const trimmed = message.trim();
    if (!trimmed) {
      return;
    }

    setLastUserMessage(trimmed);
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
      setReply(undefined);
      setStatus({ tone: "error", text: errorMessage(error) });
    }
  }

  function handleScroll(): void {
    if (scrollRef.current) {
      setShowScrollDown(shouldShowScrollDown(scrollRef.current));
    }
  }

  function scrollToBottom(behavior: ScrollBehavior = "smooth"): void {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({
        top: scrollRef.current.scrollHeight,
        behavior,
      });
    }
  }

  function scrollToSettings(): void {
    settingsRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function startNewSession(): void {
    setReply(undefined);
    setLastUserMessage("");
    setSettings((current) => ({ ...current, sessionId: "" }));
    setStatus({ tone: "idle", text: "新会话" });
  }

  return (
    <div className="mobile-chat-shell">
      <ChatHeader
        runtimeLabel={runtimeLabel}
        status={status}
        onOpenSettings={scrollToSettings}
        onNewSession={startNewSession}
      />

      <main ref={scrollRef} onScroll={handleScroll} className="chat-feed">
        <ChatBubble>我想让 Ghost-OS 通过移动端连接 Bridge，把任务交给桌面侧执行。</ChatBubble>

        <AssistantIntro
          host={host}
          config={config}
          lastTraceId={lastTraceId}
          settings={settings}
          status={status}
        />

        <BridgeSettings
          refTarget={settingsRef}
          settings={settings}
          apiToken={apiToken}
          loading={status.tone === "loading"}
          onConnect={connectBridge}
          onApiTokenChange={setApiToken}
          onSettingsChange={setSettings}
        />

        {lastUserMessage ? <ChatBubble>{lastUserMessage}</ChatBubble> : null}
        <AssistantReply reply={reply} status={status} sessionId={settings.sessionId} />
      </main>

      {showScrollDown ? (
        <button className="scroll-down-button" type="button" onClick={() => scrollToBottom()} aria-label="滚动到底部">
          <span className="ui-icon ui-icon-arrow-down" aria-hidden="true" />
        </button>
      ) : null}

      <ChatComposer
        value={message}
        disabled={!canSend}
        loading={status.tone === "loading"}
        onSubmit={sendMessage}
        onChange={setMessage}
        onOpenSettings={scrollToSettings}
      />
    </div>
  );
}

export default App;
