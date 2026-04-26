// Main web console page composing session list, messages, and input panels.

'use client';

import type { FC } from 'react';
import { ChatInput } from '@/components/ChatInput';
import { ConfigPanel } from '@/components/ConfigPanel';
import { MessageList } from '@/components/message/MessageList';
import { SessionSidebar } from '@/components/SessionSidebar';
import { useHomePageController } from '@/hooks/useHomePageController';
import { useWebLocale } from '@/lib/i18n/provider';
import { ignorePromise } from '@/lib/errors';

export const dynamic = 'force-dynamic';

const HomePage: FC = () => {
  const { copy } = useWebLocale();
  const controller = useHomePageController();

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
          />

          <section className="chat panel">
            {controller.historyLoading ? <div className="status-line info">{copy.chat.historyLoading}</div> : null}
            {controller.historySyncing && !controller.historyLoading ? <div className="status-line info">{copy.chat.historySyncing}</div> : null}
            {controller.configError && !controller.showConfig ? <div className="status-line error">{controller.configError}</div> : null}

            <MessageList
              key={controller.currentSessionId || 'draft-session'}
              committedMessages={controller.committedMessages}
              assistantMarkdownEnabled={controller.config?.assistant_markdown_enabled ?? true}
              streamingAssistantSegments={controller.streamingAssistantSegments}
              streamingThinkingText={controller.streamingThinkingText}
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
