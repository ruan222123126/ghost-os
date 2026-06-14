// Main web console page composing session list, messages, and input panels.

'use client';

import nextDynamic from 'next/dynamic';
import { useCallback, type FC } from 'react';
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
        <HomePageTopbar controller={controller} copy={copy} />

        <div className="chat-layout">
          <HomePageSidebar controller={controller} sessionAliases={sessionAliases} />
          <HomePageChatPanel controller={controller} copy={copy} currentSessionTitle={currentSessionTitle} />
        </div>
      </main>

      {controller.showConfig ? (
        <ConfigPanel
          open={controller.showConfig}
          initialTab={controller.settingsTabFromQuery ?? 'provider'}
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
      ) : null}

      {showBootLoading ? <GlobalLoadingOverlay /> : null}
    </>
  );
};

export default HomePage;

const HomePageTopbar: FC<{
  controller: ReturnType<typeof useHomePageController>;
  copy: ReturnType<typeof useWebLocale>['copy'];
}> = ({ controller, copy }) => {
  if (!controller.topStatusVisible) {
    return null;
  }

  return (
    <header className="topbar">
      <div className="topbar-actions">
        {controller.configLoading ? <span className="status-chip">{copy.chat.topbarLoadingRuntime}</span> : null}
        {controller.configError && !controller.showConfig ? <span className="status-chip status-chip-warning">{copy.chat.topbarConfigNeedsAttention}</span> : null}
      </div>
    </header>
  );
};

const HomePageSidebar: FC<{
  controller: ReturnType<typeof useHomePageController>;
  sessionAliases: ReturnType<typeof useSessionSidebarAliases>;
}> = ({ controller, sessionAliases }) => {
  return (
    <SessionSidebar
      sessions={controller.sessions}
      currentSessionId={controller.currentSessionId}
      loading={controller.sessionsLoading}
      error={controller.sessionsError}
      onSelect={controller.selectSession}
      onDelete={(id) => {
        ignorePromise(controller.deleteSession(id));
      }}
      onNewChat={controller.newChat}
      onOpenSettings={controller.openConfig}
      resolveSessionTitle={sessionAliases.resolveSessionTitle}
      renameSession={sessionAliases.renameSession}
    />
  );
};

const HomePageChatPanel: FC<{
  controller: ReturnType<typeof useHomePageController>;
  copy: ReturnType<typeof useWebLocale>['copy'];
  currentSessionTitle: string;
}> = ({ controller, copy, currentSessionTitle }) => {
  const showSystemPromptMessages = controller.config?.session_system_prompt_visible_enabled ?? true;
  const assistantMarkdownEnabled = controller.config?.assistant_markdown_enabled ?? true;
  const toolCallCompactOutputEnabled = controller.config?.tool_call_compact_output_enabled ?? false;
  const modelSelectionHandler = controller.config?.model_selection_enabled
    ? controller.selectActiveModel
    : undefined;
  const messageListView = buildMessageListProjection({
    committedMessages: controller.committedMessages,
    loading: controller.loading,
    loadingOlderHistory: controller.loadingOlderHistory,
    pendingQuestions: controller.pendingQuestions,
    showSystemPromptMessages,
    streamingAssistantSegments: controller.streamingAssistantSegments,
    activeStreamingThinkingId: controller.activeStreamingThinkingId,
    streamingItemOrder: controller.streamingItemOrder,
    streamingThinkingSegments: controller.streamingThinkingSegments,
    streamingTools: controller.streamingTools,
    toolCard: {
      compactOutputEnabled: toolCallCompactOutputEnabled,
      fallbackTitle: copy.chat.toolFallbackName,
      preparingDetails: copy.chat.toolPreparingOutput,
    },
  });

  return (
    <section className={`chat panel${controller.showEmptyHomeState ? ' is-empty-home' : ''}`}>
      {controller.showEmptyHomeState ? null : (
        <ChatSessionNotch sessionId={controller.currentSessionId} title={currentSessionTitle} />
      )}
      <ChatStatusLines controller={controller} copy={copy} />
      {controller.showEmptyHomeState ? (
        <div className="chat-empty-home">
          <ChatEmptyHero />
          {controller.chatError ? <div className="status-line error">{controller.chatError}</div> : null}
          <HomePageComposer controller={controller} onSelectModel={modelSelectionHandler} />
        </div>
      ) : (
        <>
          <MessageList
            key={controller.currentSessionId || 'draft-session'}
            view={messageListView}
            assistantMarkdownEnabled={assistantMarkdownEnabled}
            hasOlderHistory={controller.hasOlderHistory}
            loadOlderHistory={controller.loadOlderHistory}
            onAnswerQuestion={controller.answerQuestion}
            onCancelQuestion={controller.cancelQuestion}
          />
          {controller.chatError ? <div className="status-line error">{controller.chatError}</div> : null}
          <HomePageComposer controller={controller} onSelectModel={modelSelectionHandler} />
        </>
      )}
    </section>
  );
};

const HomePageComposer: FC<{
  controller: ReturnType<typeof useHomePageController>;
  onSelectModel?: ReturnType<typeof useHomePageController>['selectActiveModel'];
}> = ({ controller, onSelectModel }) => {
  return (
    <ChatInput
      loading={controller.loading}
      disabled={controller.inputDisabled || controller.savingConfig}
      awaitingQuestion={controller.hasPendingQuestion}
      modelLoading={controller.modelOptionsLoading || controller.savingConfig}
      activeModel={controller.activeModelOption}
      availableModels={controller.modelOptions}
      canStop={controller.canStop}
      onSend={controller.sendMessage}
      onStop={controller.stopCurrentRun}
      onSelectModel={onSelectModel}
    />
  );
};

const ChatStatusLines: FC<{
  controller: ReturnType<typeof useHomePageController>;
  copy: ReturnType<typeof useWebLocale>['copy'];
}> = ({ controller, copy }) => {
  if (!controller.historyLoading && !controller.historySyncing && (!controller.configError || controller.showConfig)) {
    return null;
  }

  return (
    <>
      {controller.historyLoading ? <TopLoadingBar className="chat-history-loading-bar" label={copy.chat.historyLoading} /> : null}
      {controller.historySyncing && !controller.historyLoading ? <div className="status-line info">{copy.chat.historySyncing}</div> : null}
      {controller.configError && !controller.showConfig ? <div className="status-line error">{controller.configError}</div> : null}
    </>
  );
};
