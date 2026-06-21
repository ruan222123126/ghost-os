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
import { MobileSearchPage } from "./components/MobileSearchPage";
import { MobileSettingsPanel } from "./components/MobileSettingsPanel";
import { useBodyScrollLock } from "./hooks/useBodyScrollLock";
import { useChatFeedScroll } from "./hooks/useChatFeedScroll";
import { useMobileBridge } from "./hooks/useMobileBridge";
import { useMobileSessions } from "./hooks/useMobileSessions";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "./mobileTypes";
import "./App.css";
import "./components/mobileChat/Messages.css";
import "./components/mobileChat/ToolCards.css";
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
    activateProvider,
    bridgeUrl,
    config,
    connectBridge,
    connectionStatus,
    createProvider,
    deleteProvider,
    deleteSkill,
    deleteTask,
    getFullSession,
    getSession,
    host,
    providerList,
    refreshProviders,
    refreshSkills,
    refreshTasks,
    runTaskNow,
    runningTaskId,
    searchSessions,
    sendAgentMessage,
    sessions,
    sessionsLoaded,
    setSettings,
    setTaskEnabled,
    settings,
    skillList,
    skillListError,
    switchModel,
    taskList,
    taskListError,
    status,
    updateSkill,
    updateProvider,
  } = useMobileBridge();
  const [message, setMessage] = useState("");
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isRuntimeMenuOpen, setIsRuntimeMenuOpen] = useState(false);
  const [isMoreMenuOpen, setIsMoreMenuOpen] = useState(false);
  const [pinnedHistoryIds, setPinnedHistoryIds] = useState<string[]>([]);
  const mobileSessions = useMobileSessions({
    bridgeConnected: Boolean(config),
    computerSessionSyncScope: `${settings.connectionMode}:${bridgeUrl}:${settings.pairing?.deviceId ?? ""}:${settings.pairing?.pcId ?? ""}:${settings.pairing?.signalingUrl ?? ""}`,
    getFullSession,
    getSession,
    pinnedHistoryIds,
    persistComputerSessionsEnabled: settings.persistComputerSessionsEnabled,
    sendAgentMessage,
    sessions,
    sessionsLoaded,
  });
  const displayStatus = mobileSessions.activeStatus.tone === "idle" ? status : mobileSessions.activeStatus;
  const canSend = isNonEmptyMessage(message) && mobileSessions.canSend;
  const runtimeLabel = useMemo(() => displayRuntime(config), [config]);
  const isModalOpen = isSidebarOpen || isSearchOpen || isSettingsOpen || isMoreMenuOpen;
  const hasLocalConversation = mobileSessions.hasConversation;
  const {
    handleScroll,
    registerUserMessageRow,
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
    trailingSpacerPx,
  } = useChatFeedScroll({
    messages: mobileSessions.activeMessages,
    reply: mobileSessions.activeReply,
    statusTone: mobileSessions.activeStatus.tone,
  });
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
    setIsSearchOpen(false);
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSettingsOpen(false);
    setIsSidebarOpen(true);
  }

  function openSearch(): void {
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSettingsOpen(false);
    setIsSidebarOpen(false);
    setIsSearchOpen(true);
  }

  function openSettings(): void {
    setIsSearchOpen(false);
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
        onOpenSearch={openSearch}
        onSelectHistory={(sessionId) => void selectHistory(sessionId)}
        onConnect={connectBridge}
        onOpenSettings={openSettings}
      />

      <MobileSearchPage
        open={isSearchOpen}
        bridgeConnected={Boolean(config)}
        historyItems={mobileSessions.historyItems}
        onClose={() => setIsSearchOpen(false)}
        onSelectHistory={(sessionId) => void selectHistory(sessionId)}
        onSearchSessions={searchSessions}
      />

      <div className="mobile-chat-content" aria-hidden={isModalOpen} inert={isModalOpen ? true : undefined}>
        <ChatHeader
          runtimeLabel={runtimeLabel}
          config={config}
          providerList={providerList}
          status={displayStatus}
          hasConversation={hasLocalConversation}
          runtimeMenuOpen={isRuntimeMenuOpen}
          onOpenSidebar={openSidebar}
          onToggleRuntimeMenu={() => setIsRuntimeMenuOpen((current) => !current)}
          onCloseRuntimeMenu={() => setIsRuntimeMenuOpen(false)}
          onSwitchModel={switchModel}
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
              <ChatBubble key={item.id} ref={registerUserMessageRow(item.id)}>{item.text}</ChatBubble>
            ) : (
              <AssistantReply
                key={item.id}
                reply={conversationMessageToAgentPayload(item)}
                status={assistantMessageStatus()}
              />
            ),
          )}
          <AssistantReply
            reply={mobileSessions.activeReply}
            status={displayStatus}
          />
          <div aria-hidden="true" style={{ height: trailingSpacerPx }} />
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
        config={config}
        computerSessionPersistStatus={mobileSessions.computerSessionPersistStatus}
        connectionStatus={connectionStatus}
        providerList={providerList}
        onClose={() => setIsSettingsOpen(false)}
        onConnect={connectBridge}
        onActivateProvider={activateProvider}
        onCreateProvider={createProvider}
        onDeleteProvider={deleteProvider}
        onDeleteSkill={deleteSkill}
        onDeleteTask={deleteTask}
        onRefreshProviders={refreshProviders}
        onRefreshSkills={refreshSkills}
        onRefreshTasks={refreshTasks}
        onRunTaskNow={runTaskNow}
        onSettingsChange={setSettings}
        onSetTaskEnabled={setTaskEnabled}
        onUpdateSkill={updateSkill}
        onUpdateProvider={updateProvider}
        runningTaskId={runningTaskId}
        skillListError={skillListError}
        skillList={skillList}
        taskListError={taskListError}
        taskList={taskList}
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
    tools: message.tools,
  };
}

export default App;
