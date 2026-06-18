import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import {
  AssistantIntro,
  AssistantReply,
  ChatBubble,
  ChatComposer,
  ChatHeader,
  ConversationToolPreview,
  INITIAL_HISTORY_LIST,
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

function App() {
  const {
    bridgeUrl,
    config,
    connectBridge,
    host,
    reply,
    sendAgentMessage,
    setReply,
    setSettings,
    setStatus,
    settings,
    status,
  } = useMobileBridge();
  const [message, setMessage] = useState("");
  const [lastUserMessage, setLastUserMessage] = useState("");
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isRuntimeMenuOpen, setIsRuntimeMenuOpen] = useState(false);
  const [isMoreMenuOpen, setIsMoreMenuOpen] = useState(false);
  const [historyItems, setHistoryItems] = useState<SidebarHistoryItem[]>(() => INITIAL_HISTORY_LIST);
  const [activeHistoryId, setActiveHistoryId] = useState<number | undefined>(3);

  const canSend = isNonEmptyMessage(message) && status.tone !== "loading";
  const runtimeLabel = useMemo(() => displayRuntime(config), [config]);
  const isModalOpen = isSidebarOpen || isSettingsOpen || isMoreMenuOpen;
  const hasLocalConversation = Boolean(lastUserMessage || reply);
  const { handleScroll, resetScrollDown, scrollRef, scrollToBottom, showScrollDown } = useChatFeedScroll(
    lastUserMessage,
    reply,
  );
  const activeHistoryItem = useMemo(
    () => historyItems.find((item) => item.id === activeHistoryId),
    [activeHistoryId, historyItems],
  );

  useBodyScrollLock(isModalOpen);

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

  function openPlaceholderSession(id: number, title: string): void {
    setReply(undefined);
    setLastUserMessage(title);
    setMessage("");
    setActiveHistoryId(id);
    setStatus({ tone: "idle", text: "占位会话" });
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    resetScrollDown();
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
          {hasLocalConversation && status.tone !== "error" ? <ConversationToolPreview /> : null}
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
        status={status}
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
