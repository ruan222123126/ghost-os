// Main web console page composing session list, messages, and input panels.

'use client';

import { useCallback, type FC } from 'react';
import { ChatSessionNotch } from '@/components/ChatSessionNotch';
import { ChatInput } from '@/components/ChatInput';
import { ConfigPanel } from '@/components/ConfigPanel';
import { MessageList } from '@/components/message/MessageList';
import { SessionSidebar } from '@/components/SessionSidebar';
import { useHomePageController } from '@/hooks/useHomePageController';
import { useSessionSidebarAliases } from '@/hooks/useSessionSidebarAliases';
import { useWebLocale } from '@/lib/i18n/provider';
import { ignorePromise } from '@/lib/errors';
import type { SessionMetadata } from '@/lib/types';

export const dynamic = 'force-dynamic';

const HomePage: FC = () => {
  const { copy } = useWebLocale();
  const controller = useHomePageController();
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
        {controller.topStatusVisible ? (
          <header className="topbar">
            <div className="topbar-actions">
              {controller.configLoading ? <span className="status-chip">{copy.chat.topbarLoadingRuntime}</span> : null}
              {controller.configError && !controller.showConfig ? <span className="status-chip status-chip-warning">{copy.chat.topbarConfigNeedsAttention}</span> : null}
            </div>
          </header>
        ) : null}

        <div className="chat-layout">
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

          <section className="chat panel">
            <ChatSessionNotch sessionId={controller.currentSessionId} title={currentSessionTitle} />
            {controller.historyLoading ? <div className="status-line info">{copy.chat.historyLoading}</div> : null}
            {controller.historySyncing && !controller.historyLoading ? <div className="status-line info">{copy.chat.historySyncing}</div> : null}
            {controller.configError && !controller.showConfig ? <div className="status-line error">{controller.configError}</div> : null}

            <MessageList
              key={controller.currentSessionId || 'draft-session'}
              committedMessages={controller.committedMessages}
              showSystemPromptMessages={controller.config?.session_system_prompt_visible_enabled ?? true}
              assistantMarkdownEnabled={controller.config?.assistant_markdown_enabled ?? true}
              toolCallCompactOutputEnabled={controller.config?.tool_call_compact_output_enabled ?? false}
              streamingAssistantSegments={controller.streamingAssistantSegments}
              streamingThinkingSegments={controller.streamingThinkingSegments}
              streamingItemOrder={controller.streamingItemOrder}
              streamingTools={controller.streamingTools}
              pendingQuestions={controller.pendingQuestions}
              loading={controller.loading}
              loadingOlderHistory={controller.loadingOlderHistory}
              hasOlderHistory={controller.hasOlderHistory}
              loadOlderHistory={controller.loadOlderHistory}
              onAnswerQuestion={controller.answerQuestion}
              onCancelQuestion={controller.cancelQuestion}
            />

            {controller.chatError ? <div className="status-line error">{controller.chatError}</div> : null}

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
              onSelectModel={controller.config?.model_selection_enabled ? controller.selectActiveModel : undefined}
            />
          </section>
        </div>
      </main>

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
    </>
  );
};

export default HomePage;
