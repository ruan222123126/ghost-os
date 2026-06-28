// Main web console page composing session list, messages, and input panels.

'use client';

import nextDynamic from 'next/dynamic';
import { useCallback, useMemo, type FC } from 'react';
import { ChatEmptyHero } from '@/components/ChatEmptyHero';
import { ChatSessionNotch } from '@/components/ChatSessionNotch';
import { ChatInput } from '@/components/ChatInput';
import type { ConfigPanelProps } from '@/components/ConfigPanel';
import { GlobalLoadingOverlay } from '@/components/GlobalLoadingOverlay';
import type { MessageListProps } from '@/components/message/types';
import { SessionSidebar } from '@/components/SessionSidebar';
import { TopLoadingBar } from '@/components/TopLoadingBar';
import { useHomePageController } from '@/hooks/useHomePageController';
import { useInitialLoadingOverlay } from '@/hooks/useInitialLoadingOverlay';
import { useSessionSidebarAliases } from '@/hooks/useSessionSidebarAliases';
import { useWebLocale } from '@/lib/i18n/provider';
import { ignorePromise } from '@/lib/errors';
import { buildMessageListProjection } from '@/lib/chat-view/messageRows';
import type { SessionMetadata } from '@/lib/types';

export const dynamic = 'force-dynamic';

type HomePageController = ReturnType<typeof useHomePageController>;
type WebCopy = ReturnType<typeof useWebLocale>['copy'];

const ConfigPanel = nextDynamic<ConfigPanelProps>(
  () => import('@/components/ConfigPanel').then((mod) => mod.ConfigPanel),
  { ssr: false },
);

function MessageListLoading(): JSX.Element {
  return <div className="messages is-empty" aria-live="polite" />;
}

const MessageList = nextDynamic<MessageListProps>(
  () => import('@/components/message/MessageList').then((mod) => mod.MessageList),
  {
    loading: MessageListLoading,
    ssr: false,
  },
);

const HomePage: FC = () => {
  const { copy } = useWebLocale();
  const controller = useHomePageController();
  const showBootLoading = useInitialLoadingOverlay({
    ready: !controller.configLoading && !controller.modelOptionsLoading && !controller.sessionsLoading,
  });
  const resolveDefaultSessionTitleByID = useCallback((sessionID: string) => {
    return copy.chat.sidebarSessionTitle(sessionID.trim().slice(0, 8));
  }, [copy.chat]);
  const resolveDefaultSessionTitle = useCallback((session: SessionMetadata) => {
    return resolveDefaultSessionTitleByID(session.id);
  }, [resolveDefaultSessionTitleByID]);
  const sessionAliases = useSessionSidebarAliases({
    sessions: controller.sessions,
    resolveDefaultTitle: resolveDefaultSessionTitle,
  });
  const currentSessionTitle = controller.currentSessionId
    ? sessionAliases.resolveSessionTitleByID(controller.currentSessionId)
      || resolveDefaultSessionTitleByID(controller.currentSessionId)
    : '';

  return (
    <>
      <div className="ambient ambient-a" aria-hidden="true" />
      <div className="ambient ambient-b" aria-hidden="true" />

      <main className="app-shell">
        <HomePageTopbar
          copy={copy}
          topStatusVisible={controller.topStatusVisible}
          configLoading={controller.configLoading}
          configError={controller.configError}
          showConfig={controller.showConfig}
        />

        <div className="chat-layout">
          <HomePageSidebar
            sessions={controller.sessions}
            currentSessionId={controller.currentSessionId}
            backgroundCompletedSessionIds={controller.backgroundCompletedSessionIds}
            loading={controller.sessionsLoading}
            error={controller.sessionsError}
            onSelect={controller.selectSession}
            onDelete={controller.deleteSession}
            onNewChat={controller.newChat}
            onOpenSettings={controller.openConfig}
            sessionAliases={sessionAliases}
          />
          <HomePageChatPanel
            copy={copy}
            currentSessionTitle={currentSessionTitle}
            config={controller.config}
            currentSessionId={controller.currentSessionId}
            showEmptyHomeState={controller.showEmptyHomeState}
            chatError={controller.chatError}
            committedMessages={controller.committedMessages}
            loading={controller.loading}
            loadingOlderHistory={controller.loadingOlderHistory}
            pendingQuestions={controller.pendingQuestions}
            streamingAssistantSegments={controller.streamingAssistantSegments}
            activeStreamingThinkingId={controller.activeStreamingThinkingId}
            streamingItemOrder={controller.streamingItemOrder}
            streamingThinkingSegments={controller.streamingThinkingSegments}
            streamingTools={controller.streamingTools}
            historyLoading={controller.historyLoading}
            configError={controller.configError}
            showConfig={controller.showConfig}
            hasOlderHistory={controller.hasOlderHistory}
            loadOlderHistory={controller.loadOlderHistory}
            answerQuestion={controller.answerQuestion}
            cancelQuestion={controller.cancelQuestion}
            inputDisabled={controller.inputDisabled}
            savingConfig={controller.savingConfig}
            hasPendingQuestion={controller.hasPendingQuestion}
            modelOptionsLoading={controller.modelOptionsLoading}
            activeModelOption={controller.activeModelOption}
            modelOptions={controller.modelOptions}
            canStop={controller.canStop}
            sendMessage={controller.sendMessage}
            stopCurrentRun={controller.stopCurrentRun}
            selectActiveModel={controller.selectActiveModel}
          />
        </div>
      </main>

      <ConfigPanel
        open={controller.showConfig}
        initialTab={controller.settingsTabFromQuery ?? 'general'}
        loading={controller.configLoading}
        saving={controller.savingConfig}
        config={controller.config}
        error={controller.configError}
        onClose={controller.closeConfig}
        onOpenWorkflowCreate={controller.openWorkflowCreate}
        onOpenWorkflowEdit={controller.openWorkflowEdit}
        onSave={controller.saveConfig}
        onReload={controller.refreshConfig}
      />

      {showBootLoading ? <GlobalLoadingOverlay /> : null}
    </>
  );
};

export default HomePage;

const HomePageTopbar: FC<{
  copy: WebCopy;
  topStatusVisible: HomePageController['topStatusVisible'];
  configLoading: HomePageController['configLoading'];
  configError: HomePageController['configError'];
  showConfig: HomePageController['showConfig'];
}> = ({ copy, topStatusVisible, configLoading, configError, showConfig }) => {
  if (!topStatusVisible) {
    return null;
  }

  return (
    <header className="topbar">
      <div className="topbar-actions">
        {configLoading ? <span className="status-chip">{copy.chat.topbarLoadingRuntime}</span> : null}
        {configError && !showConfig ? <span className="status-chip status-chip-warning">{copy.chat.topbarConfigNeedsAttention}</span> : null}
      </div>
    </header>
  );
};

const HomePageSidebar: FC<{
  sessions: HomePageController['sessions'];
  currentSessionId: HomePageController['currentSessionId'];
  backgroundCompletedSessionIds: HomePageController['backgroundCompletedSessionIds'];
  loading: HomePageController['sessionsLoading'];
  error: HomePageController['sessionsError'];
  onSelect: HomePageController['selectSession'];
  onDelete: HomePageController['deleteSession'];
  onNewChat: HomePageController['newChat'];
  onOpenSettings: HomePageController['openConfig'];
  sessionAliases: ReturnType<typeof useSessionSidebarAliases>;
}> = ({
  sessions,
  currentSessionId,
  backgroundCompletedSessionIds,
  loading,
  error,
  onSelect,
  onDelete,
  onNewChat,
  onOpenSettings,
  sessionAliases,
}) => {
  return (
    <SessionSidebar
      sessions={sessions}
      currentSessionId={currentSessionId}
      backgroundCompletedSessionIds={backgroundCompletedSessionIds}
      loading={loading}
      error={error}
      onSelect={onSelect}
      onDelete={(id) => {
        ignorePromise(onDelete(id));
      }}
      onNewChat={onNewChat}
      onOpenSettings={onOpenSettings}
      resolveSessionTitle={sessionAliases.resolveSessionTitle}
      renameSession={sessionAliases.renameSession}
    />
  );
};

const HomePageChatPanel: FC<{
  copy: WebCopy;
  currentSessionTitle: string;
  config: HomePageController['config'];
  currentSessionId: HomePageController['currentSessionId'];
  showEmptyHomeState: HomePageController['showEmptyHomeState'];
  chatError: HomePageController['chatError'];
  committedMessages: HomePageController['committedMessages'];
  loading: HomePageController['loading'];
  loadingOlderHistory: HomePageController['loadingOlderHistory'];
  pendingQuestions: HomePageController['pendingQuestions'];
  streamingAssistantSegments: HomePageController['streamingAssistantSegments'];
  activeStreamingThinkingId: HomePageController['activeStreamingThinkingId'];
  streamingItemOrder: HomePageController['streamingItemOrder'];
  streamingThinkingSegments: HomePageController['streamingThinkingSegments'];
  streamingTools: HomePageController['streamingTools'];
  historyLoading: HomePageController['historyLoading'];
  configError: HomePageController['configError'];
  showConfig: HomePageController['showConfig'];
  hasOlderHistory: HomePageController['hasOlderHistory'];
  loadOlderHistory: HomePageController['loadOlderHistory'];
  answerQuestion: HomePageController['answerQuestion'];
  cancelQuestion: HomePageController['cancelQuestion'];
  inputDisabled: HomePageController['inputDisabled'];
  savingConfig: HomePageController['savingConfig'];
  hasPendingQuestion: HomePageController['hasPendingQuestion'];
  modelOptionsLoading: HomePageController['modelOptionsLoading'];
  activeModelOption: HomePageController['activeModelOption'];
  modelOptions: HomePageController['modelOptions'];
  canStop: HomePageController['canStop'];
  sendMessage: HomePageController['sendMessage'];
  stopCurrentRun: HomePageController['stopCurrentRun'];
  selectActiveModel: HomePageController['selectActiveModel'];
}> = ({
  copy,
  currentSessionTitle,
  config,
  currentSessionId,
  showEmptyHomeState,
  chatError,
  committedMessages,
  loading,
  loadingOlderHistory,
  pendingQuestions,
  streamingAssistantSegments,
  activeStreamingThinkingId,
  streamingItemOrder,
  streamingThinkingSegments,
  streamingTools,
  historyLoading,
  configError,
  showConfig,
  hasOlderHistory,
  loadOlderHistory,
  answerQuestion,
  cancelQuestion,
  inputDisabled,
  savingConfig,
  hasPendingQuestion,
  modelOptionsLoading,
  activeModelOption,
  modelOptions,
  canStop,
  sendMessage,
  stopCurrentRun,
  selectActiveModel,
}) => {
  const showSystemPromptMessages = config?.session_system_prompt_visible_enabled ?? true;
  const assistantMarkdownEnabled = config?.assistant_markdown_enabled ?? true;
  const modelSelectionHandler = config?.model_selection_enabled
    ? selectActiveModel
    : undefined;
  const composerNode = (
    <HomePageComposer
      canEnableCodexMode={Boolean(config)}
      loading={loading}
      inputDisabled={inputDisabled}
      savingConfig={savingConfig}
      hasPendingQuestion={hasPendingQuestion}
      modelOptionsLoading={modelOptionsLoading}
      activeModelOption={activeModelOption}
      modelOptions={modelOptions}
      canStop={canStop}
      sendMessage={sendMessage}
      stopCurrentRun={stopCurrentRun}
      onSelectModel={modelSelectionHandler}
    />
  );
  const messageListView = useMemo(() => {
    return buildMessageListProjection({
      committedMessages,
      loading,
      loadingOlderHistory,
      pendingQuestions,
      showSystemPromptMessages,
      streamingAssistantSegments,
      activeStreamingThinkingId,
      streamingItemOrder,
      streamingThinkingSegments,
      streamingTools,
      toolCard: {
        fallbackTitle: copy.chat.toolFallbackName,
        preparingDetails: copy.chat.toolPreparingOutput,
      },
    });
  }, [
    activeStreamingThinkingId,
    committedMessages,
    copy.chat.toolFallbackName,
    copy.chat.toolPreparingOutput,
    loading,
    loadingOlderHistory,
    pendingQuestions,
    showSystemPromptMessages,
    streamingAssistantSegments,
    streamingItemOrder,
    streamingThinkingSegments,
    streamingTools,
  ]);

  return (
    <section className={`chat panel${showEmptyHomeState ? ' is-empty-home' : ''}`}>
      {showEmptyHomeState ? null : (
        <ChatSessionNotch sessionId={currentSessionId} title={currentSessionTitle} />
      )}
      <ChatStatusLines
        copy={copy}
        historyLoading={historyLoading}
        configError={configError}
        showConfig={showConfig}
      />
      {showEmptyHomeState ? (
        <div className="chat-empty-home">
          <ChatEmptyHero />
          {chatError ? <div className="status-line error">{chatError}</div> : null}
          {composerNode}
        </div>
      ) : (
        <>
          <MessageList
            key={currentSessionId || 'draft-session'}
            view={messageListView}
            assistantMarkdownEnabled={assistantMarkdownEnabled}
            hasOlderHistory={hasOlderHistory}
            loadOlderHistory={loadOlderHistory}
            onAnswerQuestion={answerQuestion}
            onCancelQuestion={cancelQuestion}
          />
          {chatError ? <div className="status-line error">{chatError}</div> : null}
          {composerNode}
        </>
      )}
    </section>
  );
};

const HomePageComposer: FC<{
  canEnableCodexMode: boolean;
  loading: HomePageController['loading'];
  inputDisabled: HomePageController['inputDisabled'];
  savingConfig: HomePageController['savingConfig'];
  hasPendingQuestion: HomePageController['hasPendingQuestion'];
  modelOptionsLoading: HomePageController['modelOptionsLoading'];
  activeModelOption: HomePageController['activeModelOption'];
  modelOptions: HomePageController['modelOptions'];
  canStop: HomePageController['canStop'];
  sendMessage: HomePageController['sendMessage'];
  stopCurrentRun: HomePageController['stopCurrentRun'];
  onSelectModel?: HomePageController['selectActiveModel'];
}> = ({
  canEnableCodexMode,
  loading,
  inputDisabled,
  savingConfig,
  hasPendingQuestion,
  modelOptionsLoading,
  activeModelOption,
  modelOptions,
  canStop,
  sendMessage,
  stopCurrentRun,
  onSelectModel,
}) => {
  return (
    <ChatInput
      loading={loading}
      canEnableCodexMode={canEnableCodexMode}
      disabled={inputDisabled || savingConfig}
      awaitingQuestion={hasPendingQuestion}
      modelLoading={modelOptionsLoading || savingConfig}
      activeModel={activeModelOption}
      availableModels={modelOptions}
      canStop={canStop}
      onSend={sendMessage}
      onStop={stopCurrentRun}
      onSelectModel={onSelectModel}
    />
  );
};

const ChatStatusLines: FC<{
  copy: WebCopy;
  historyLoading: HomePageController['historyLoading'];
  configError: HomePageController['configError'];
  showConfig: HomePageController['showConfig'];
}> = ({ copy, historyLoading, configError, showConfig }) => {
  if (!historyLoading && (!configError || showConfig)) {
    return null;
  }

  return (
    <>
      {historyLoading ? <TopLoadingBar className="chat-history-loading-bar" label={copy.chat.historyLoading} /> : null}
      {configError && !showConfig ? <div className="status-line error">{configError}</div> : null}
    </>
  );
};
