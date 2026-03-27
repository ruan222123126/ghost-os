// Main web console page composing session list, messages, and input panels.

'use client';

import type { FC } from 'react';
import { useCallback, useState } from 'react';
import { ChatInput } from '@/components/ChatInput';
import { ConfigPanel } from '@/components/ConfigPanel';
import { MessageList } from '@/components/message/MessageList';
import { SessionSidebar } from '@/components/SessionSidebar';
import { useBridgeChat } from '@/hooks/chat/useBridgeChat';
import { useBridgeConfig } from '@/hooks/useBridgeConfig';
import { useSessions } from '@/hooks/useSessions';
import { ignorePromise } from '@/lib/errors';
import type { ChatSendInput } from '@/lib/types';

const HomePage: FC = () => {
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
    messages,
    loading,
    historyLoading,
    chatError,
    hasPendingQuestion,
    canStop,
    sendChatMessage,
    stopCurrentRun,
    answerQuestion,
    cancelQuestion,
    loadSessionHistory,
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
            {configError && !showConfig ? <div className="status-line error">{configError}</div> : null}

            <MessageList
              messages={messages}
              loading={loading}
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
        loading={configLoading}
        saving={savingConfig}
        config={config}
        error={configError}
        onClose={() => setShowConfig(false)}
        onSave={saveConfig}
        onReload={refreshConfig}
      />
    </>
  );
};

export default HomePage;
