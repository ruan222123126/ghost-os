import { invoke } from "@tauri-apps/api/core";
import { ArrowDown } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import {
  AssistantIntro,
  AssistantReply,
  ChatBubble,
  ChatComposer,
  ChatHeader,
  ConversationPlaceholder,
  INITIAL_HISTORY_LIST,
  MobileSidebar,
  MoreActionSheet,
} from "./components/MobileChatHome";
import { MobileSettingsPanel } from "./components/MobileSettingsPanel";
import type { SidebarHistoryItem } from "./components/MobileChatHome";
import type { AgentPayload, ConfigPayload, HostProfile, StatusMessage, StoredSettings } from "./mobileTypes";
import "./App.css";
import "./App.overlays.css";

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

function hasTauriRuntime(): boolean {
  return "__TAURI_INTERNALS__" in window;
}

function App() {
  const [settings, setSettings] = useState<StoredSettings>(() => loadSettings());
  const [apiToken] = useState("");
  const [message, setMessage] = useState("");
  const [host, setHost] = useState<HostProfile>();
  const [config, setConfig] = useState<ConfigPayload>();
  const [reply, setReply] = useState<AgentPayload>();
  const [lastUserMessage, setLastUserMessage] = useState("");
  const [showScrollDown, setShowScrollDown] = useState(false);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isRuntimeMenuOpen, setIsRuntimeMenuOpen] = useState(false);
  const [isMoreMenuOpen, setIsMoreMenuOpen] = useState(false);
  const [historyItems, setHistoryItems] = useState<SidebarHistoryItem[]>(() => INITIAL_HISTORY_LIST);
  const [activeHistoryId, setActiveHistoryId] = useState<number | undefined>(3);
  const [status, setStatus] = useState<StatusMessage>({
    tone: "idle",
    text: "未连接",
  });

  const scrollRef = useRef<HTMLElement>(null);
  const bridgeUrl = useMemo(() => normalizeBridgeUrl(settings.bridgeUrl), [settings.bridgeUrl]);
  const canSend = isNonEmptyMessage(message) && status.tone !== "loading";
  const runtimeLabel = useMemo(() => displayRuntime(config), [config]);
  const isModalOpen = isSidebarOpen || isSettingsOpen || isMoreMenuOpen;
  const hasLocalConversation = Boolean(lastUserMessage || reply);
  const activeHistoryItem = useMemo(
    () => historyItems.find((item) => item.id === activeHistoryId),
    [activeHistoryId, historyItems],
  );

  useEffect(() => {
    if (!hasTauriRuntime()) {
      return;
    }

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

  useEffect(() => {
    document.body.style.overflow = isModalOpen ? "hidden" : "auto";
    return () => {
      document.body.style.overflow = "auto";
    };
  }, [isModalOpen]);

  async function requestBridge<TPayload>(
    action: string,
    params: Record<string, unknown>,
  ): Promise<TPayload> {
    const traceId = createTraceId(`android-${action.toLowerCase()}`);

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
    setShowScrollDown(false);
    if (scrollRef.current) {
      scrollRef.current.scrollTo({
        top: scrollRef.current.scrollHeight,
        behavior,
      });
    }
  }

  function openSidebar(): void {
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSettingsOpen(false);
    setIsSidebarOpen(true);
  }

  function openSettings(): void {
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    setIsSettingsOpen(true);
  }

  function openPlaceholderSession(id: number, title: string): void {
    setReply(undefined);
    setLastUserMessage(title);
    setMessage("");
    setActiveHistoryId(id);
    setStatus({ tone: "idle", text: "占位会话" });
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    setShowScrollDown(false);
  }

  function startNewSession(): void {
    setReply(undefined);
    setLastUserMessage("");
    setMessage("");
    setActiveHistoryId(undefined);
    setSettings((current) => ({ ...current, sessionId: "" }));
    setStatus({ tone: "idle", text: "新会话" });
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    setShowScrollDown(false);
  }

  function clearLocalConversation(): void {
    setReply(undefined);
    setLastUserMessage("");
    setActiveHistoryId(undefined);
    setStatus({ tone: "idle", text: "本地消息已清空" });
    setIsMoreMenuOpen(false);
    setShowScrollDown(false);
  }

  function toggleActiveHistoryPin(): void {
    if (!activeHistoryItem) {
      return;
    }

    setHistoryItems((current) =>
      current.map((item) => (item.id === activeHistoryItem.id ? { ...item, pinned: !item.pinned } : item)),
    );
    setIsMoreMenuOpen(false);
  }

  return (
    <div className="mobile-chat-shell">
      <MobileSidebar
        open={isSidebarOpen}
        host={host}
        config={config}
        settings={settings}
        historyItems={historyItems}
        activeHistoryId={activeHistoryId}
        onClose={() => setIsSidebarOpen(false)}
        onNewSession={startNewSession}
        onSelectSession={openPlaceholderSession}
        onConnect={connectBridge}
        onOpenSettings={openSettings}
      />

      <div className="mobile-chat-content" aria-hidden={isModalOpen} inert={isModalOpen ? true : undefined}>
        <ChatHeader
          runtimeLabel={runtimeLabel}
          config={config}
          status={status}
          bridgeUrl={bridgeUrl}
          hasConversation={hasLocalConversation}
          runtimeMenuOpen={isRuntimeMenuOpen}
          onOpenSidebar={openSidebar}
          onToggleRuntimeMenu={() => setIsRuntimeMenuOpen((current) => !current)}
          onCloseRuntimeMenu={() => setIsRuntimeMenuOpen(false)}
          onOpenSettings={openSettings}
          onOpenMoreMenu={() => {
            setIsRuntimeMenuOpen(false);
            setIsMoreMenuOpen(true);
          }}
          onNewSession={startNewSession}
        />

        <main
          ref={scrollRef}
          onScroll={handleScroll}
          className={`chat-feed ${hasLocalConversation ? "" : "is-empty"}`}
        >
          {!hasLocalConversation ? (
            <AssistantIntro onSelectSuggestion={setMessage} />
          ) : null}

          {lastUserMessage ? <ChatBubble>{lastUserMessage}</ChatBubble> : null}
          <AssistantReply reply={reply} status={status} sessionId={settings.sessionId} />
          {hasLocalConversation ? <ConversationPlaceholder /> : null}
        </main>

        {showScrollDown ? (
          <button className="scroll-down-button" type="button" onClick={() => scrollToBottom()} aria-label="滚动到底部">
            <ArrowDown className="ui-icon" aria-hidden="true" strokeWidth={1.75} />
          </button>
        ) : null}

        <ChatComposer
          value={message}
          disabled={!canSend}
          loading={status.tone === "loading"}
          onSubmit={sendMessage}
          onChange={setMessage}
          onOpenSettings={openSettings}
        />
      </div>

      <MobileSettingsPanel open={isSettingsOpen} onClose={() => setIsSettingsOpen(false)} />

      <MoreActionSheet
        open={isMoreMenuOpen}
        hasLocalConversation={hasLocalConversation}
        canTogglePin={Boolean(activeHistoryItem)}
        isPinned={activeHistoryItem?.pinned ?? false}
        onClose={() => setIsMoreMenuOpen(false)}
        onTogglePin={toggleActiveHistoryPin}
        onClearConversation={clearLocalConversation}
      />
    </div>
  );
}

export default App;
