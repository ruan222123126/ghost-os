import { startTransition, useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties, KeyboardEvent } from "react";
import {
  AssistantIntro,
  ChatHeader,
  ConversationMessageList,
  MobileChatComposer,
  MobileSidebar,
  MoreActionSheet,
  ScrollDownButton,
} from "./components/MobileChatHome";
import type { MobileChatComposerHandle } from "./components/MobileChatHome";
import { MobileConnectionPanel } from "./components/MobileConnectionPanel";
import { MobileSearchPage } from "./components/MobileSearchPage";
import { MobileSettingsPanel } from "./components/MobileSettingsPanel";
import { useBodyScrollLock } from "./hooks/useBodyScrollLock";
import { useChatFeedScroll } from "./hooks/useChatFeedScroll";
import { useMobileBridge } from "./hooks/useMobileBridge";
import { useMobileSessionCompletionNotifications } from "./hooks/useMobileSessionCompletionNotifications";
import { useMobileSessions } from "./hooks/useMobileSessions";
import { normalizeCodexModel } from "./lib/codexModels";
import type { SessionCompletionEvent } from "./lib/mobileSessionRunTracker";
import type {
  AgentModeSelection,
  AgentRuntimeType,
  ChatSelectedSkill,
  ConfigPayload,
  ProviderListPayload,
  SessionRuntimeSelection,
  StoredSettings,
} from "./mobileTypes";
import "markstream-react/index.css";
import "./App.css";
import "./components/mobileChat/Messages.css";
import "./components/mobileChat/ToolCards.css";
import "./App.overlays.css";

const SIDEBAR_CLOSE_DEFER_MS = 320;
const COMPLETION_NOTIFICATION_STACK_LIMIT = 8;

interface CompletionNotification {
  id: string;
  title: string;
}

function displayRuntime(
  agentRuntime: AgentRuntimeType,
  config: ReturnType<typeof useMobileBridge>["config"],
  codexModel: string,
): string {
  if (agentRuntime === "codex") {
    return codexModel ? `Codex / ${codexModel}` : "Codex";
  }
  if (config?.provider && config.model) {
    return `${config.provider} / ${config.model}`;
  }
  return config?.provider || config?.model || "Bridge Runtime";
}

function resolveEffectiveAgentRuntime(
  agentRuntime: AgentRuntimeType,
  agentMode: AgentModeSelection,
): AgentRuntimeType {
  return agentMode === null ? agentRuntime : "codex";
}

function resolveLocalProvider(providerList: ProviderListPayload | undefined, settings: StoredSettings) {
  return providerList?.providers.find((provider) => provider.provider_id === settings.localProviderId)
    ?? providerList?.providers.find((provider) => !provider.deleted_at);
}

function buildLocalRuntimeConfig(
  providerList: ProviderListPayload | undefined,
  settings: StoredSettings,
): ConfigPayload | undefined {
  const provider = resolveLocalProvider(providerList, settings);
  if (!provider) {
    return undefined;
  }
  return {
    model: settings.localModel?.trim() || provider.models?.[0]?.trim() || "",
    provider: provider.name,
  };
}

function App() {
  const {
    activateProvider,
    appendSessionMessages,
    approveExternalAgent,
    bridgeUrl,
    config,
    codexModelCatalog,
    codexModelCatalogError,
    connectBridge,
    connectionStatus,
    createProvider,
    deleteOrchestration,
    deleteProvider,
    deleteSkill,
    deleteTask,
    getFullSession,
    getSession,
    getSessionRunStates,
    host,
    orchestrationList,
    orchestrationListError,
    providerList,
    refreshOrchestrations,
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
    setStatus,
    setOrchestrationEnabled,
    setTaskEnabled,
    settings,
    skillList,
    skillListError,
    stopAgentRun,
    switchModel,
    switchRuntimeSelection,
    taskList,
    taskListError,
    status,
    updateSkill,
    updateExternalCodexPermissionMode,
    updateProvider,
  } = useMobileBridge();
  const [agentRuntime, setAgentRuntime] = useState<AgentRuntimeType>("ghost");
  const [agentMode, setAgentMode] = useState<AgentModeSelection>(null);
  const [codexModel, setCodexModel] = useState<string>("");
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [isConnectionOpen, setIsConnectionOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isRuntimeMenuOpen, setIsRuntimeMenuOpen] = useState(false);
  const [isMoreMenuOpen, setIsMoreMenuOpen] = useState(false);
  const [pinnedHistoryIds, setPinnedHistoryIds] = useState<string[]>([]);
  const [completionNotifications, setCompletionNotifications] = useState<CompletionNotification[]>([]);
  const composerRef = useRef<MobileChatComposerHandle>(null);
  const virtualScrollToBottomRef = useRef<(() => void) | null>(null);
  const pendingSelectHistoryTimeoutRef = useRef<number | null>(null);
  const localRuntimeConfig = useMemo(() => buildLocalRuntimeConfig(providerList, settings), [providerList, settings]);
  const chatConfig = settings.remoteExecutionEnabled ? config : localRuntimeConfig;
  const chatProviderList = providerList;
  const activeCodexModel = normalizeCodexModel(codexModel, codexModelCatalog);
  const effectiveAgentRuntime = resolveEffectiveAgentRuntime(agentRuntime, agentMode);
  const effectiveRuntimeConfig = effectiveAgentRuntime === "codex" ? config : chatConfig;
  const connectionScope = `${settings.connectionMode}:${bridgeUrl}:${settings.pairing?.deviceId ?? ""}:${settings.pairing?.pcId ?? ""}:${settings.pairing?.signalingUrl ?? ""}`;
  const sendAgentMessageForRuntime = useCallback(
    (options: Parameters<typeof sendAgentMessage>[0]) => sendAgentMessage({
      ...options,
      agentRuntime: effectiveAgentRuntime,
      mode: agentMode === "plan" ? "plan" : undefined,
      codexModel: effectiveAgentRuntime === "codex" ? activeCodexModel : undefined,
    }),
    [activeCodexModel, agentMode, effectiveAgentRuntime, sendAgentMessage],
  );
  const stopAgentRunForRuntime = useCallback(
    (input: Parameters<typeof stopAgentRun>[0]) => stopAgentRun({ ...input, agentRuntime: effectiveAgentRuntime }),
    [effectiveAgentRuntime, stopAgentRun],
  );
  const applySessionRuntimeSelection = useCallback(
    async (selection: SessionRuntimeSelection | null | undefined): Promise<void> => {
      if (!selection) {
        return;
      }
      if (selection.runtime === "codex" || selection.mode === "plan") {
        setAgentRuntime("codex");
        setAgentMode(selection.mode === "plan" ? "plan" : "normal");
        if (selection.runtime === "codex" && selection.model?.trim()) {
          setCodexModel(normalizeCodexModel(selection.model, codexModelCatalog));
        }
        return;
      }

      setAgentRuntime("ghost");
      setAgentMode(null);
      await switchRuntimeSelection(selection);
    },
    [codexModelCatalog, switchRuntimeSelection],
  );
  const mobileSessions = useMobileSessions({
    bridgeConnected: Boolean(config),
    appendSessionMessages,
    computerSessionSyncScope: connectionScope,
    getFullSession,
    getSession,
    onSessionRuntimeSelection: applySessionRuntimeSelection,
    pinnedHistoryIds,
    persistComputerSessionsEnabled: settings.persistComputerSessionsEnabled,
    sendAvailable: effectiveAgentRuntime === "codex"
      ? Boolean(config)
      : settings.remoteExecutionEnabled
      ? Boolean(config)
      : Boolean(localRuntimeConfig?.provider && localRuntimeConfig?.model),
    sendAgentMessage: sendAgentMessageForRuntime,
    sessions,
    sessionsLoaded,
    stopAgentRun: stopAgentRunForRuntime,
  });
  const handleSessionCompleted = useCallback((event: SessionCompletionEvent): void => {
    setCompletionNotifications((current) => [
      ...current.filter((notification) => notification.id !== event.sessionId),
      { id: event.sessionId, title: event.title.trim() || "该会话" },
    ].slice(-COMPLETION_NOTIFICATION_STACK_LIMIT));
  }, []);
  const knownSessionIds = useMemo(
    () => [...new Set([...sessions.map((session) => session.id), ...mobileSessions.historyItems.map((item) => item.id)])],
    [mobileSessions.historyItems, sessions],
  );
  useMobileSessionCompletionNotifications({
    activeSessionId: mobileSessions.activeSessionId,
    connected: connectionStatus.tone === "success",
    connectionScope,
    getSessionRunStates,
    knownSessionIds,
    liveRunningSessions: mobileSessions.liveRunningSessions,
    onInAppCompletion: handleSessionCompleted,
    onStatus: setStatus,
    selectSession: mobileSessions.selectSession,
    sessionsLoaded,
  });
  const displayStatus = mobileSessions.activeStatus.tone === "idle" ? status : mobileSessions.activeStatus;
  const supportsComposerSkills = Boolean(config) && (effectiveAgentRuntime === "codex" || settings.remoteExecutionEnabled);
  const runtimeLabel = useMemo(
    () => displayRuntime(effectiveAgentRuntime, effectiveRuntimeConfig, activeCodexModel),
    [activeCodexModel, effectiveAgentRuntime, effectiveRuntimeConfig],
  );
  const isModalOpen = isSidebarOpen || isSearchOpen || isConnectionOpen || isSettingsOpen || isMoreMenuOpen;
  const hasLocalConversation = mobileSessions.hasConversation;
  const showEmptyIntro = !hasLocalConversation && !mobileSessions.loadingSessionMessages;
  const showTopLoadingBar = mobileSessions.loadingSessionMessages || mobileSessions.loadingOlderHistory;
  const {
    handleScroll,
    handleUserScrollEnd,
    handleUserScrollIntent,
    handleUserScrollStart,
    historySentinelRef,
    registerUserMessageRow,
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
    trailingSpacerRef,
    trailingSpacerPx,
  } = useChatFeedScroll({
    hasOlderHistory: mobileSessions.hasOlderHistory,
    loadingOlderHistory: mobileSessions.loadingOlderHistory,
    messages: mobileSessions.activeMessages,
    onLoadOlderHistory: mobileSessions.loadOlderHistory,
    postSendFocusRequest: mobileSessions.postSendFocusRequest,
    reply: mobileSessions.activeReply,
    sessionId: mobileSessions.activeSessionId,
    statusTone: mobileSessions.activeStatus.tone,
  });
  const selectSessionRef = useRef(mobileSessions.selectSession);
  const sendMessageRef = useRef(mobileSessions.sendMessage);
  const stopCurrentRunRef = useRef(mobileSessions.stopCurrentRun);
  const refreshSkillsRef = useRef(refreshSkills);
  const activeHistoryItem = useMemo(
    () => mobileSessions.historyItems.find((item) => item.id === mobileSessions.activeSessionId),
    [mobileSessions.activeSessionId, mobileSessions.historyItems],
  );

  useBodyScrollLock(isModalOpen);

  useEffect(() => {
    selectSessionRef.current = mobileSessions.selectSession;
  }, [mobileSessions.selectSession]);

  useEffect(() => {
    sendMessageRef.current = mobileSessions.sendMessage;
    stopCurrentRunRef.current = mobileSessions.stopCurrentRun;
    refreshSkillsRef.current = refreshSkills;
  }, [mobileSessions.sendMessage, mobileSessions.stopCurrentRun, refreshSkills]);

  const sendComposerMessage = useCallback(
    (text: string, selectedSkill?: ChatSelectedSkill) => sendMessageRef.current(text, selectedSkill),
    [],
  );
  const stopComposerRun = useCallback(() => stopCurrentRunRef.current(), []);
  const refreshComposerSkills = useCallback(() => refreshSkillsRef.current(), []);

  useEffect(() => {
    return () => {
      clearPendingSelectHistory();
    };
  }, []);

  function switchAgentRuntime(runtime: AgentRuntimeType): void {
    setAgentRuntime(runtime);
    if (runtime !== "codex") {
      setAgentMode(null);
    }
  }

  function selectHistory(sessionId: string): void {
    composerRef.current?.reset();
    const shouldDeferSelection = isSidebarOpen;
    setIsSidebarOpen(false);
    clearPendingSelectHistory();
    if (shouldDeferSelection) {
      pendingSelectHistoryTimeoutRef.current = window.setTimeout(() => {
        pendingSelectHistoryTimeoutRef.current = null;
        startHistorySelection(sessionId);
      }, SIDEBAR_CLOSE_DEFER_MS);
      return;
    }
    startHistorySelection(sessionId);
  }

  function dismissCompletionNotification(sessionId: string): void {
    setCompletionNotifications((current) => current.filter((notification) => notification.id !== sessionId));
  }

  function openCompletionNotification(sessionId: string): void {
    dismissCompletionNotification(sessionId);
    selectHistory(sessionId);
  }

  function startHistorySelection(sessionId: string): void {
    startTransition(() => {
      void selectSessionRef.current(sessionId);
    });
  }

  function clearPendingSelectHistory(): void {
    if (pendingSelectHistoryTimeoutRef.current === null) {
      return;
    }
    window.clearTimeout(pendingSelectHistoryTimeoutRef.current);
    pendingSelectHistoryTimeoutRef.current = null;
  }

  const setChatFeedRef = useCallback((node: HTMLElement | null) => {
    scrollRef.current = node;
  }, [scrollRef]);

  function openSidebar(): void {
    setIsSearchOpen(false);
    setIsConnectionOpen(false);
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSettingsOpen(false);
    setIsSidebarOpen(true);
  }

  function openSearch(): void {
    setIsConnectionOpen(false);
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSettingsOpen(false);
    setIsSidebarOpen(false);
    setIsSearchOpen(true);
  }

  function openSettings(): void {
    setIsSearchOpen(false);
    setIsConnectionOpen(false);
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    setIsSettingsOpen(true);
  }

  function openConnection(): void {
    setIsSearchOpen(false);
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    setIsSettingsOpen(false);
    setIsConnectionOpen(true);
  }

  function startNewSession(): void {
    composerRef.current?.reset();
    mobileSessions.startNewSession();
    setIsConnectionOpen(false);
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    resetScrollDown();
  }

  function clearLocalConversation(): void {
    composerRef.current?.reset();
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
        onSelectHistory={selectHistory}
        onConnect={connectBridge}
        onOpenSettings={openSettings}
      />

      <MobileSearchPage
        open={isSearchOpen}
        bridgeConnected={Boolean(config)}
        historyItems={mobileSessions.historyItems}
        onClose={() => setIsSearchOpen(false)}
        onSelectHistory={selectHistory}
        onSearchSessions={searchSessions}
      />

      <CompletionNotificationStack
        notifications={completionNotifications}
        onDismiss={dismissCompletionNotification}
        onOpen={openCompletionNotification}
      />

      <div
        className={`mobile-chat-content ${isSidebarOpen ? "is-sidebar-open" : ""}`}
        aria-hidden={isModalOpen}
        inert={isModalOpen ? true : undefined}
      >
        <ChatHeader
          runtimeLabel={runtimeLabel}
          agentRuntime={effectiveAgentRuntime}
          codexModel={activeCodexModel}
          codexModelCatalog={codexModelCatalog}
          codexModelCatalogError={codexModelCatalogError}
          config={effectiveRuntimeConfig}
          codexPermissionMode={config?.external_codex_permission_mode}
          providerList={chatProviderList}
          status={displayStatus}
          hasConversation={hasLocalConversation}
          runtimeMenuOpen={isRuntimeMenuOpen}
          onOpenSidebar={openSidebar}
          onToggleRuntimeMenu={() => setIsRuntimeMenuOpen((current) => !current)}
          onCloseRuntimeMenu={() => setIsRuntimeMenuOpen(false)}
          onSwitchModel={switchModel}
          onSwitchAgentRuntime={switchAgentRuntime}
          onSwitchCodexModel={setCodexModel}
          onOpenConnection={openConnection}
          onOpenMoreMenu={() => {
            setIsRuntimeMenuOpen(false);
            setIsMoreMenuOpen(true);
          }}
          onNewSession={startNewSession}
        />

        {showTopLoadingBar ? <MobileTopLoadingBar label="消息加载中" /> : null}

        <main
          ref={setChatFeedRef}
          onScroll={handleScroll}
          onPointerCancel={handleUserScrollEnd}
          onPointerDown={handleUserScrollStart}
          onPointerUp={handleUserScrollEnd}
          onTouchCancel={handleUserScrollEnd}
          onTouchEnd={handleUserScrollEnd}
          onTouchMove={handleUserScrollIntent}
          onTouchStart={handleUserScrollStart}
          onWheel={handleUserScrollIntent}
          className={`chat-feed ${showEmptyIntro ? "is-empty" : ""}`}
        >
          {showEmptyIntro ? (
            <AssistantIntro onSelectSuggestion={(value) => composerRef.current?.setDraft(value)} />
          ) : null}

          {!showEmptyIntro ? (
            <div ref={historySentinelRef} className="chat-feed-history-sentinel" aria-hidden="true" />
          ) : null}

          <ConversationMessageList
            messages={mobileSessions.activeMessages}
            onApproveExternalAgent={approveExternalAgent}
            postSendFocusRequest={mobileSessions.postSendFocusRequest}
            registerUserMessageRow={registerUserMessageRow}
            reply={mobileSessions.activeReply}
            scrollElementRef={scrollRef}
            scrollToBottomRef={virtualScrollToBottomRef}
            status={displayStatus}
          />
          <div
            ref={trailingSpacerRef}
            aria-hidden="true"
            className="chat-feed-trailing-spacer"
            style={{ minHeight: trailingSpacerPx }}
          />
        </main>

        {showScrollDown ? (
          <ScrollDownButton
            onClick={() => scrollToBottom("smooth", virtualScrollToBottomRef.current ?? undefined)}
          />
        ) : null}

        <MobileChatComposer
          ref={composerRef}
          agentMode={agentMode}
          canEnableCodexMode={Boolean(config)}
          canSend={mobileSessions.canSend}
          canStop={(settings.remoteExecutionEnabled || effectiveAgentRuntime === "codex") && mobileSessions.canStop}
          loading={mobileSessions.activeStatus.tone === "loading"}
          skills={supportsComposerSkills ? skillList : undefined}
          onChangeAgentMode={setAgentMode}
          onRefreshSkills={supportsComposerSkills ? refreshComposerSkills : undefined}
          onSend={sendComposerMessage}
          onStop={(settings.remoteExecutionEnabled || effectiveAgentRuntime === "codex") ? stopComposerRun : undefined}
        />
      </div>

      <MobileConnectionPanel
        open={isConnectionOpen}
        settings={settings}
        computerSessionPersistStatus={mobileSessions.computerSessionPersistStatus}
        connectionStatus={connectionStatus}
        onClose={() => setIsConnectionOpen(false)}
        onConnect={connectBridge}
        onSettingsChange={setSettings}
      />

      <MobileSettingsPanel
        open={isSettingsOpen}
        config={chatConfig}
        codexPermissionMode={config?.external_codex_permission_mode}
        connectionStatus={connectionStatus}
        providerList={providerList}
        onClose={() => setIsSettingsOpen(false)}
        onActivateProvider={activateProvider}
        onCreateProvider={createProvider}
        onDeleteOrchestration={deleteOrchestration}
        onDeleteProvider={deleteProvider}
        onDeleteSkill={deleteSkill}
        onDeleteTask={deleteTask}
        onRefreshProviders={refreshProviders}
        onRefreshOrchestrations={refreshOrchestrations}
        onRefreshSkills={refreshSkills}
        onRefreshTasks={refreshTasks}
        onRunTaskNow={runTaskNow}
        onSetOrchestrationEnabled={setOrchestrationEnabled}
        onSetTaskEnabled={setTaskEnabled}
        onUpdateSkill={updateSkill}
        onUpdateCodexPermission={updateExternalCodexPermissionMode}
        onUpdateProvider={updateProvider}
        runningTaskId={runningTaskId}
        skillListError={skillListError}
        skillList={skillList}
        orchestrationList={orchestrationList}
        orchestrationListError={orchestrationListError}
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

function MobileTopLoadingBar(props: { label: string }) {
  return (
    <div className="mobile-top-loading-bar" role="status" aria-label={props.label}>
      <div className="mobile-top-loading-bar-fill" aria-hidden="true" />
    </div>
  );
}

function CompletionNotificationStack(props: {
  notifications: CompletionNotification[];
  onDismiss: (sessionId: string) => void;
  onOpen: (sessionId: string) => void;
}) {
  if (props.notifications.length === 0) {
    return null;
  }

  return (
    <div className="completion-notification-stack" aria-live="polite">
      {props.notifications.map((notification, index) => {
        const depth = props.notifications.length - 1 - index;
        return (
          <div
            key={notification.id}
            className="completion-notification-card"
            role="button"
            style={{
              left: `${depth * 8}px`,
              top: `${depth * 8}px`,
              transform: `scale(${1 - depth * 0.025})`,
              zIndex: props.notifications.length - depth,
            } as CSSProperties}
            tabIndex={0}
            onClick={() => props.onOpen(notification.id)}
            onKeyDown={(event: KeyboardEvent<HTMLDivElement>) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                props.onOpen(notification.id);
              }
            }}
          >
            <span>{notification.title}会话已完成</span>
            <button
              className="completion-notification-close"
              type="button"
              aria-label="关闭"
              onClick={(event) => {
                event.stopPropagation();
                props.onDismiss(notification.id);
              }}
            >
              <span aria-hidden="true">×</span>
            </button>
          </div>
        );
      })}
    </div>
  );
}

export default App;
