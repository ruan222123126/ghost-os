import { startTransition, useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import {
  AssistantIntro,
  ChatComposer,
  ChatHeader,
  ConversationMessageList,
  MobileSidebar,
  MoreActionSheet,
  ScrollDownButton,
} from "./components/MobileChatHome";
import { MobileConnectionPanel } from "./components/MobileConnectionPanel";
import { MobileSearchPage } from "./components/MobileSearchPage";
import { MobileSettingsPanel } from "./components/MobileSettingsPanel";
import { useBodyScrollLock } from "./hooks/useBodyScrollLock";
import { useChatFeedScroll } from "./hooks/useChatFeedScroll";
import { useMobileBridge } from "./hooks/useMobileBridge";
import { useMobileSessions } from "./hooks/useMobileSessions";
import { DEFAULT_CODEX_MODEL, normalizeCodexModel } from "./lib/codexModels";
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

function isNonEmptyMessage(value: string): boolean {
  return value.trim().length > 0;
}

const SIDEBAR_CLOSE_DEFER_MS = 320;

function displayRuntime(
  agentRuntime: AgentRuntimeType,
  config: ReturnType<typeof useMobileBridge>["config"],
  codexModel: string = DEFAULT_CODEX_MODEL,
): string {
  if (agentRuntime === "codex") {
    return `Codex / ${normalizeCodexModel(codexModel)}`;
  }
  if (config?.provider && config.model) {
    return `${config.provider} / ${config.model}`;
  }
  return config?.provider || config?.model || "Bridge Runtime";
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
    connectBridge,
    connectionStatus,
    createProvider,
    deleteOrchestration,
    deleteProvider,
    deleteSkill,
    deleteTask,
    getFullSession,
    getSession,
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
  const [message, setMessage] = useState("");
  const [selectedSkill, setSelectedSkill] = useState<ChatSelectedSkill | null>(null);
  const [agentRuntime, setAgentRuntime] = useState<AgentRuntimeType>("ghost");
  const [agentMode, setAgentMode] = useState<AgentModeSelection>(null);
  const [codexModel, setCodexModel] = useState<string>(DEFAULT_CODEX_MODEL);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [isConnectionOpen, setIsConnectionOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isRuntimeMenuOpen, setIsRuntimeMenuOpen] = useState(false);
  const [isMoreMenuOpen, setIsMoreMenuOpen] = useState(false);
  const [pinnedHistoryIds, setPinnedHistoryIds] = useState<string[]>([]);
  const pendingSelectHistoryTimeoutRef = useRef<number | null>(null);
  const localRuntimeConfig = useMemo(() => buildLocalRuntimeConfig(providerList, settings), [providerList, settings]);
  const chatConfig = settings.remoteExecutionEnabled ? config : localRuntimeConfig;
  const chatProviderList = providerList;
  const activeCodexModel = normalizeCodexModel(codexModel);
  const effectiveAgentRuntime = agentMode === "plan" ? "ghost" : agentRuntime;
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
      if (selection.runtime === "codex") {
        setAgentRuntime("codex");
        setAgentMode("normal");
        if (selection.model?.trim()) {
          setCodexModel(normalizeCodexModel(selection.model));
        }
        return;
      }

      setAgentRuntime("ghost");
      setAgentMode(selection.mode === "plan" ? "plan" : null);
      await switchRuntimeSelection(selection);
    },
    [switchRuntimeSelection],
  );
  const mobileSessions = useMobileSessions({
    bridgeConnected: Boolean(config),
    appendSessionMessages,
    computerSessionSyncScope: `${settings.connectionMode}:${bridgeUrl}:${settings.pairing?.deviceId ?? ""}:${settings.pairing?.pcId ?? ""}:${settings.pairing?.signalingUrl ?? ""}`,
    getFullSession,
    getSession,
    onSessionRuntimeSelection: applySessionRuntimeSelection,
    pinnedHistoryIds,
    persistComputerSessionsEnabled: settings.persistComputerSessionsEnabled,
    sendAvailable: agentMode === "plan"
      ? Boolean(config)
      : agentRuntime === "codex"
      ? Boolean(config)
      : settings.remoteExecutionEnabled
      ? Boolean(config)
      : Boolean(localRuntimeConfig?.provider && localRuntimeConfig?.model),
    sendAgentMessage: sendAgentMessageForRuntime,
    sessions,
    sessionsLoaded,
    stopAgentRun: stopAgentRunForRuntime,
  });
  const displayStatus = mobileSessions.activeStatus.tone === "idle" ? status : mobileSessions.activeStatus;
  const supportsComposerSkills = Boolean(config) && (agentMode === "plan" || agentRuntime === "codex" || settings.remoteExecutionEnabled);
  const canSubmit = isNonEmptyMessage(message) || selectedSkill !== null;
  const runtimeLabel = useMemo(
    () => displayRuntime(agentRuntime, agentRuntime === "codex" ? config : chatConfig, activeCodexModel),
    [activeCodexModel, agentRuntime, chatConfig, config],
  );
  const isModalOpen = isSidebarOpen || isSearchOpen || isConnectionOpen || isSettingsOpen || isMoreMenuOpen;
  const hasLocalConversation = mobileSessions.hasConversation;
  const showEmptyIntro = !hasLocalConversation && !mobileSessions.loadingSessionMessages;
  const showTopLoadingBar = mobileSessions.loadingSessionMessages || mobileSessions.loadingOlderHistory;
  const {
    handleScroll,
    historySentinelRef,
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
  } = useChatFeedScroll({
    hasOlderHistory: mobileSessions.hasOlderHistory,
    loadingOlderHistory: mobileSessions.loadingOlderHistory,
    messages: mobileSessions.activeMessages,
    onLoadOlderHistory: mobileSessions.loadOlderHistory,
    postSendScrollRequest: mobileSessions.postSendScrollRequest,
    reply: mobileSessions.activeReply,
    sessionId: mobileSessions.activeSessionId,
    statusTone: mobileSessions.activeStatus.tone,
  });
  const selectSessionRef = useRef(mobileSessions.selectSession);
  const activeHistoryItem = useMemo(
    () => mobileSessions.historyItems.find((item) => item.id === mobileSessions.activeSessionId),
    [mobileSessions.activeSessionId, mobileSessions.historyItems],
  );

  useBodyScrollLock(isModalOpen);

  useEffect(() => {
    selectSessionRef.current = mobileSessions.selectSession;
  }, [mobileSessions.selectSession]);

  useEffect(() => {
    return () => {
      clearPendingSelectHistory();
    };
  }, []);

  useEffect(() => {
    if (!supportsComposerSkills && selectedSkill !== null) {
      setSelectedSkill(null);
    }
  }, [selectedSkill, supportsComposerSkills]);

  async function sendMessage(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const trimmed = message.trim();
    if (!trimmed && !selectedSkill) {
      return;
    }

    const previousMessage = message;
    const previousSkill = selectedSkill;
    setMessage("");
    setSelectedSkill(null);
    const sent = await mobileSessions.sendMessage(trimmed, selectedSkill);
    if (!sent) {
      setMessage(previousMessage);
      setSelectedSkill(previousSkill);
    }
  }

  function selectHistory(sessionId: string): void {
    setMessage("");
    setSelectedSkill(null);
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
    setMessage("");
    setSelectedSkill(null);
    mobileSessions.startNewSession();
    setIsConnectionOpen(false);
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    resetScrollDown();
  }

  function clearLocalConversation(): void {
    setSelectedSkill(null);
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

      <div
        className={`mobile-chat-content ${isSidebarOpen ? "is-sidebar-open" : ""}`}
        aria-hidden={isModalOpen}
        inert={isModalOpen ? true : undefined}
      >
        <ChatHeader
          runtimeLabel={runtimeLabel}
          agentRuntime={agentRuntime}
          codexModel={activeCodexModel}
          config={agentRuntime === "codex" ? config : chatConfig}
          codexPermissionMode={config?.external_codex_permission_mode}
          providerList={chatProviderList}
          status={displayStatus}
          hasConversation={hasLocalConversation}
          runtimeMenuOpen={isRuntimeMenuOpen}
          onOpenSidebar={openSidebar}
          onToggleRuntimeMenu={() => setIsRuntimeMenuOpen((current) => !current)}
          onCloseRuntimeMenu={() => setIsRuntimeMenuOpen(false)}
          onSwitchModel={switchModel}
          onSwitchAgentRuntime={setAgentRuntime}
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
          className={`chat-feed ${showEmptyIntro ? "is-empty" : "is-reverse-flow"}`}
        >
          {showEmptyIntro ? (
            <AssistantIntro onSelectSuggestion={setMessage} />
          ) : null}

          <ConversationMessageList
            messages={mobileSessions.activeMessages}
            onApproveExternalAgent={approveExternalAgent}
            reply={mobileSessions.activeReply}
            status={displayStatus}
          />

          {!showEmptyIntro ? (
            <div ref={historySentinelRef} className="chat-feed-history-sentinel" aria-hidden="true" />
          ) : null}
        </main>

        {showScrollDown ? <ScrollDownButton onClick={() => scrollToBottom()} /> : null}

        <ChatComposer
          agentMode={agentMode}
          canEnableCodexMode={Boolean(config)}
          canSubmit={canSubmit}
          disabled={!mobileSessions.canSend}
          value={message}
          canStop={(settings.remoteExecutionEnabled || agentRuntime === "codex" || agentMode === "plan") && mobileSessions.canStop}
          loading={mobileSessions.activeStatus.tone === "loading"}
          selectedSkill={selectedSkill}
          skills={supportsComposerSkills ? skillList : undefined}
          onClearSelectedSkill={() => setSelectedSkill(null)}
          onRefreshSkills={supportsComposerSkills ? refreshSkills : undefined}
          onSelectSkill={supportsComposerSkills ? (skill) => setSelectedSkill({ id: skill.id, name: skill.name }) : undefined}
          onChangeAgentMode={setAgentMode}
          onSubmit={sendMessage}
          onStop={(settings.remoteExecutionEnabled || agentRuntime === "codex" || agentMode === "plan") ? async () => {
            await mobileSessions.stopCurrentRun();
          } : undefined}
          onChange={setMessage}
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

export default App;
