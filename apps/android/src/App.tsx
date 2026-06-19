import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import {
  AssistantIntro,
  AssistantReply,
  ChatBubble,
  ChatComposer,
  ChatHeader,
  MobileSidebar,
  MoreActionSheet,
  ScrollDownButton,
} from "./components/MobileChatHome";
import { MobileSettingsPanel } from "./components/MobileSettingsPanel";
import type { SidebarHistoryItem } from "./components/MobileChatHome";
import { useBodyScrollLock } from "./hooks/useBodyScrollLock";
import { useChatFeedScroll } from "./hooks/useChatFeedScroll";
import { useMobileBridge } from "./hooks/useMobileBridge";
import "./App.css";
import "./components/mobileChat/Messages.css";
import "./App.overlays.css";

function isNonEmptyMessage(value: string): boolean {
  return value.trim().length > 0;
}

function displayRuntime(config: ReturnType<typeof useMobileBridge>["config"]): string {
  if (config?.provider && config.model) {
    return `${config.provider} / ${config.model}`;
  }
  return config?.provider || config?.model || "Bridge Runtime";
}

function sessionFallbackTitle(sessionId: string): string {
  const shortId = sessionId.trim().slice(0, 8);
  return shortId ? `会话 ${shortId}` : "新会话";
}

function App() {
  const {
    bridgeUrl,
    config,
    connectBridge,
    connectionStatus,
    host,
    providerList,
    reply,
    sendAgentMessage,
    sessions,
    setReply,
    setSettings,
    setStatus,
    settings,
    switchModel,
    status,
  } = useMobileBridge();
  const [message, setMessage] = useState("");
  const [lastUserMessage, setLastUserMessage] = useState("");
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isRuntimeMenuOpen, setIsRuntimeMenuOpen] = useState(false);
  const [isMoreMenuOpen, setIsMoreMenuOpen] = useState(false);
  const [pinnedHistoryIds, setPinnedHistoryIds] = useState<string[]>([]);
  const [activeHistoryId, setActiveHistoryId] = useState<string | undefined>();

  const canSend = isNonEmptyMessage(message) && status.tone !== "loading";
  const runtimeLabel = useMemo(() => displayRuntime(config), [config]);
  const isModalOpen = isSidebarOpen || isSettingsOpen || isMoreMenuOpen;
  const hasLocalConversation = Boolean(lastUserMessage || reply);
  const { handleScroll, resetScrollDown, scrollRef, scrollToBottom, showScrollDown } = useChatFeedScroll(
    lastUserMessage,
    reply,
  );
  const historyItems = useMemo<SidebarHistoryItem[]>(() => {
    const pinned = new Set(pinnedHistoryIds);
    return sessions.map((session) => ({
      id: session.id,
      pinned: pinned.has(session.id),
      title: session.title.trim() || sessionFallbackTitle(session.id),
      updatedAt: session.updated_at,
    }));
  }, [pinnedHistoryIds, sessions]);
  const activeHistoryItem = useMemo(
    () => historyItems.find((item) => item.id === activeHistoryId),
    [activeHistoryId, historyItems],
  );

  useBodyScrollLock(isModalOpen);

  useEffect(() => {
    const sessionId = settings.sessionId.trim();
    if (sessionId) {
      setActiveHistoryId(sessionId);
    }
  }, [settings.sessionId]);

  async function sendMessage(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const trimmed = message.trim();
    if (!trimmed) {
      return;
    }

    setLastUserMessage(trimmed);
    const didSend = await sendAgentMessage({ message: trimmed });
    if (didSend) {
      setMessage("");
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
    resetScrollDown();
  }

  function clearLocalConversation(): void {
    setReply(undefined);
    setLastUserMessage("");
    setActiveHistoryId(undefined);
    setStatus({ tone: "idle", text: "本地消息已清空" });
    setIsMoreMenuOpen(false);
    resetScrollDown();
  }

  function toggleActiveHistoryPin(): void {
    if (!activeHistoryItem) {
      return;
    }

    setPinnedHistoryIds((current) =>
      current.includes(activeHistoryItem.id)
        ? current.filter((id) => id !== activeHistoryItem.id)
        : [...current, activeHistoryItem.id],
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
        onConnect={connectBridge}
        onOpenSettings={openSettings}
      />

      <div className="mobile-chat-content" aria-hidden={isModalOpen} inert={isModalOpen ? true : undefined}>
        <ChatHeader
          runtimeLabel={runtimeLabel}
          config={config}
          providerList={providerList}
          status={status}
          bridgeUrl={bridgeUrl}
          hasConversation={hasLocalConversation}
          runtimeMenuOpen={isRuntimeMenuOpen}
          onOpenSidebar={openSidebar}
          onToggleRuntimeMenu={() => setIsRuntimeMenuOpen((current) => !current)}
          onCloseRuntimeMenu={() => setIsRuntimeMenuOpen(false)}
          onSwitchModel={switchModel}
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
        </main>

        {showScrollDown ? <ScrollDownButton onClick={() => scrollToBottom()} /> : null}

        <ChatComposer
          value={message}
          disabled={!canSend}
          loading={status.tone === "loading"}
          onSubmit={sendMessage}
          onChange={setMessage}
          onOpenSettings={openSettings}
        />
      </div>

      <MobileSettingsPanel
        open={isSettingsOpen}
        settings={settings}
        connectionStatus={connectionStatus}
        onClose={() => setIsSettingsOpen(false)}
        onConnect={connectBridge}
        onSettingsChange={setSettings}
      />

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
