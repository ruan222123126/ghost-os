import { useMemo, useState } from "react";
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
import { useBodyScrollLock } from "./hooks/useBodyScrollLock";
import { useChatFeedScroll } from "./hooks/useChatFeedScroll";
import { useMobileBridge } from "./hooks/useMobileBridge";
import { useMobileSessions } from "./hooks/useMobileSessions";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "./mobileTypes";
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

function assistantMessageStatus(): StatusMessage {
  return { tone: "success", text: "回复已返回" };
}

function App() {
  const {
    bridgeUrl,
    config,
    connectBridge,
    connectionStatus,
    getSession,
    host,
    providerList,
    sendAgentMessage,
    sessions,
    sessionsLoaded,
    setSettings,
    settings,
    switchModel,
    status,
  } = useMobileBridge();
  const [message, setMessage] = useState("");
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isRuntimeMenuOpen, setIsRuntimeMenuOpen] = useState(false);
  const [isMoreMenuOpen, setIsMoreMenuOpen] = useState(false);
  const [pinnedHistoryIds, setPinnedHistoryIds] = useState<string[]>([]);
  const mobileSessions = useMobileSessions({
    bridgeConnected: Boolean(config),
    getSession,
    pinnedHistoryIds,
    sendAgentMessage,
    sessions,
    sessionsLoaded,
  });
  const displayStatus = mobileSessions.activeStatus.tone === "idle" ? status : mobileSessions.activeStatus;
  const canSend = isNonEmptyMessage(message) && mobileSessions.canSend;
  const runtimeLabel = useMemo(() => displayRuntime(config), [config]);
  const isModalOpen = isSidebarOpen || isSettingsOpen || isMoreMenuOpen;
  const hasLocalConversation = mobileSessions.hasConversation;
  const conversationScrollKey = useMemo(
    () => mobileSessions.activeMessages.map((item) => `${item.id}:${item.text.length}`).join("|"),
    [mobileSessions.activeMessages],
  );
  const { handleScroll, resetScrollDown, scrollRef, scrollToBottom, showScrollDown } = useChatFeedScroll(
    conversationScrollKey,
    mobileSessions.activeReply,
  );
  const activeHistoryItem = useMemo(
    () => mobileSessions.historyItems.find((item) => item.id === mobileSessions.activeSessionId),
    [mobileSessions.activeSessionId, mobileSessions.historyItems],
  );

  useBodyScrollLock(isModalOpen);

  async function sendMessage(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const trimmed = message.trim();
    if (!trimmed) {
      return;
    }

    const sent = await mobileSessions.sendMessage(trimmed);
    if (sent) {
      setMessage("");
    }
  }

  async function selectHistory(sessionId: string): Promise<void> {
    setMessage("");
    setIsSidebarOpen(false);
    await mobileSessions.selectSession(sessionId);
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
    setMessage("");
    mobileSessions.startNewSession();
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    resetScrollDown();
  }

  function clearLocalConversation(): void {
    mobileSessions.clearCurrentConversation();
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
        historyItems={mobileSessions.historyItems}
        activeHistoryId={mobileSessions.activeSessionId}
        onClose={() => setIsSidebarOpen(false)}
        onNewSession={startNewSession}
        onSelectHistory={(sessionId) => void selectHistory(sessionId)}
        onConnect={connectBridge}
        onOpenSettings={openSettings}
      />

      <div className="mobile-chat-content" aria-hidden={isModalOpen} inert={isModalOpen ? true : undefined}>
        <ChatHeader
          runtimeLabel={runtimeLabel}
          config={config}
          providerList={providerList}
          status={displayStatus}
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

          {mobileSessions.activeMessages.map((item) =>
            item.role === "user" ? (
              <ChatBubble key={item.id}>{item.text}</ChatBubble>
            ) : (
              <AssistantReply
                key={item.id}
                reply={conversationMessageToAgentPayload(item)}
                status={assistantMessageStatus()}
                sessionId={item.sessionId || mobileSessions.activeSessionId || ""}
              />
            ),
          )}
          <AssistantReply
            reply={mobileSessions.activeReply}
            status={displayStatus}
            sessionId={mobileSessions.activeSessionId || ""}
          />
        </main>

        {showScrollDown ? <ScrollDownButton onClick={() => scrollToBottom()} /> : null}

        <ChatComposer
          value={message}
          disabled={!canSend}
          loading={mobileSessions.activeStatus.tone === "loading"}
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

function conversationMessageToAgentPayload(message: MobileConversationMessage): AgentPayload {
  return {
    message: message.text,
    session_ended: false,
    session_id: message.sessionId ?? "",
    thinking: message.thinking,
  };
}

export default App;
