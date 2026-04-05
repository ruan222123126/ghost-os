// Main web console page composing session list, messages, and input panels.

'use client';

import type { FC } from 'react';
import { useCallback, useEffect, useState } from 'react';
import { ChatInput } from '@/components/ChatInput';
import { ConfigPanel } from '@/components/ConfigPanel';
import { MessageList } from '@/components/message/MessageList';
import { SessionSidebar } from '@/components/SessionSidebar';
import { useBridgeChat } from '@/hooks/chat/useBridgeChat';
import { useBridgeConfig } from '@/hooks/useBridgeConfig';
import { useSessions } from '@/hooks/useSessions';
import { ignorePromise } from '@/lib/errors';
import { parseSettingsQuery, stripSettingsQuery } from '@/lib/settingsQuery';
import type { ChatSendInput } from '@/lib/types';
import { useRouter } from 'next/navigation';

export const dynamic = 'force-dynamic';

const HomePage: FC = () => {
  const router = useRouter();
  const [queryString, setQueryString] = useState('');
  const settingsTabFromQuery = parseSettingsQuery(queryString);
  const [showConfig, setShowConfig] = useState(false);
  const {
    sessions,
    currentSessionId,
    loading: sessionsLoading,
    error: sessionsError,
    loadSessions,
    deleteSession,
    createNewSession,
    setCurrentSessionId,
  } = useSessions();
  const {
    committedMessages,
    streamingAssistantSegments,
    streamingItemOrder,
    streamingTools,
    pendingQuestions,
    loading,
    historySyncing,
    historyLoading,
    loadingOlderHistory,
    chatError,
    hasPendingQuestion,
    hasOlderHistory,
    canStop,
    sendChatMessage,
    stopCurrentRun,
    answerQuestion,
    cancelQuestion,
    loadSessionHistory,
    loadOlderHistory,
    clearMessages,
  } = useBridgeChat({
    currentSessionId,
    onSessionResolved: setCurrentSessionId,
  });
  const {
    config,
    configLoading,
    savingConfig,
    configError,
    modelOptionsLoading,
    activeModelOption,
    modelOptions,
    saveConfig,
    selectActiveModel,
    refreshConfig,
  } = useBridgeConfig({ autoRefresh: !showConfig });
  const inputDisabled = configLoading || historyLoading || !config || hasPendingQuestion;
  const topStatusVisible = configLoading || (Boolean(configError) && !showConfig);

  useEffect(() => {
    const syncLocationSearch = () => {
      setQueryString(window.location.search);
    };
    syncLocationSearch();
    window.addEventListener('popstate', syncLocationSearch);

    return () => {
      window.removeEventListener('popstate', syncLocationSearch);
    };
  }, []);

  useEffect(() => {
    if (settingsTabFromQuery === 'tasks') {
      setShowConfig(true);
    }
  }, [settingsTabFromQuery]);

  const handleSendChatMessage = useCallback(
    async (input: ChatSendInput) => {
      await sendChatMessage(input);
      await loadSessions();
    },
    [loadSessions, sendChatMessage]
  );

  const handleSelectSession = useCallback(
    (id: string) => {
      setCurrentSessionId(id);
      ignorePromise(loadSessionHistory(id));
    },
    [loadSessionHistory, setCurrentSessionId]
  );

  const handleDeleteSession = useCallback(
    async (id: string) => {
      await deleteSession(id);
      if (currentSessionId === id) {
        clearMessages();
      }
    },
    [clearMessages, currentSessionId, deleteSession]
  );

  const handleNewChat = useCallback(() => {
    createNewSession();
    clearMessages();
  }, [clearMessages, createNewSession]);

  const handleCloseConfig = useCallback(() => {
    setShowConfig(false);
    if (!settingsTabFromQuery) {
      return;
    }
    const nextQuery = stripSettingsQuery(queryString);
    setQueryString(nextQuery);
    router.replace(nextQuery.length > 0 ? `/${nextQuery}` : '/');
  }, [queryString, router, settingsTabFromQuery]);

  const handleOpenWorkflowCreate = useCallback(() => {
    setShowConfig(false);
    router.push('/workflow/new');
  }, [router]);

  const handleOpenWorkflowEdit = useCallback((taskID: string) => {
    setShowConfig(false);
    router.push(`/workflow/${encodeURIComponent(taskID)}`);
  }, [router]);

  return (
    <>
      <div className="ambient ambient-a" aria-hidden="true" />
      <div className="ambient ambient-b" aria-hidden="true" />

      <main className="app-shell">
        {topStatusVisible ? (
          <header className="topbar">
            <div className="topbar-actions">
              {configLoading ? <span className="status-chip">Loading runtime…</span> : null}
              {configError && !showConfig ? <span className="status-chip status-chip-warning">Config needs attention</span> : null}
            </div>
          </header>
        ) : null}

        <div className="chat-layout">
          <SessionSidebar
            sessions={sessions}
            currentSessionId={currentSessionId}
            loading={sessionsLoading}
            error={sessionsError}
            onSelect={handleSelectSession}
            onDelete={(id) => {
              ignorePromise(handleDeleteSession(id));
            }}
            onNewChat={handleNewChat}
            onOpenSettings={() => setShowConfig(true)}
          />

          <section className="chat panel">
            {historyLoading ? <div className="status-line info">Loading session history…</div> : null}
            {historySyncing && !historyLoading ? <div className="status-line info">Syncing latest messages…</div> : null}
            {configError && !showConfig ? <div className="status-line error">{configError}</div> : null}

            <MessageList
              key={currentSessionId || 'draft-session'}
              committedMessages={committedMessages}
              streamingAssistantSegments={streamingAssistantSegments}
              streamingItemOrder={streamingItemOrder}
              streamingTools={streamingTools}
              pendingQuestions={pendingQuestions}
              loading={loading}
              loadingOlderHistory={loadingOlderHistory}
              hasOlderHistory={hasOlderHistory}
              loadOlderHistory={loadOlderHistory}
              onAnswerQuestion={answerQuestion}
              onCancelQuestion={cancelQuestion}
            />

            {chatError ? <div className="status-line error">{chatError}</div> : null}

            <ChatInput
              loading={loading}
              disabled={inputDisabled || savingConfig}
              awaitingQuestion={hasPendingQuestion}
              modelLoading={modelOptionsLoading || savingConfig}
              activeModel={activeModelOption}
              availableModels={modelOptions}
              canStop={canStop}
              onSend={handleSendChatMessage}
              onStop={stopCurrentRun}
              onSelectModel={config?.model_selection_enabled ? selectActiveModel : undefined}
            />
          </section>
        </div>
      </main>

      <ConfigPanel
        open={showConfig}
        initialTab={settingsTabFromQuery ?? 'provider'}
        loading={configLoading}
        saving={savingConfig}
        config={config}
        error={configError}
        onClose={handleCloseConfig}
        onOpenWorkflowCreate={handleOpenWorkflowCreate}
        onOpenWorkflowEdit={(task) => handleOpenWorkflowEdit(task.id)}
        onSave={saveConfig}
        onReload={refreshConfig}
      />
    </>
  );
};

export default HomePage;
