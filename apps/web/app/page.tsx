// Main web console page composing session list, messages, and input panels.

'use client';

import type { FC } from 'react';
import { useCallback, useState } from 'react';
import { ChatInput } from '@/components/ChatInput';
import { ConfigPanel } from '@/components/ConfigPanel';
import { MessageList } from '@/components/MessageList';
import { ModelSelector } from '@/components/ModelSelector';
import { SessionSidebar } from '@/components/SessionSidebar';
import { useBridgeChat } from '@/hooks/useBridgeChat';
import { useBridgeConfig } from '@/hooks/useBridgeConfig';
import { useSessions } from '@/hooks/useSessions';
import { ignorePromise } from '@/lib/errors';

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
    sendChatMessage,
    answerQuestion,
    cancelQuestion,
    loadSessionHistory,
    clearMessages,
  } = useBridgeChat({
    currentSessionId,
    onSessionResolved: setCurrentSessionId,
  });
  const { config, configLoading, savingConfig, configError, modelValue, saveConfig, selectModel, refreshConfig } = useBridgeConfig({ autoRefresh: !showConfig });
  const inputDisabled = configLoading || historyLoading || !config || hasPendingQuestion;

  const handleSendChatMessage = useCallback(
    async (message: string) => {
      await sendChatMessage(message);
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
    <main className="mx-auto flex min-h-screen w-full max-w-[1220px] gap-4 px-4 pb-6 pt-8 sm:px-6 sm:pt-10">
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
      />

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="ui-panel animate-riseSoft mb-4 p-4 sm:p-5">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1 className="text-2xl font-semibold tracking-tight text-app-text">Ghost-OS Web Console</h1>
              <p className="mt-1 text-sm text-app-muted">Drive the existing bridge agent from browser with minimal latency.</p>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              {configLoading && <span className="text-xs text-app-muted">Loading runtime config...</span>}
              <ModelSelector model={modelValue} disabled={savingConfig || configLoading || !config} onChange={selectModel} />
              <button
                type="button"
                onClick={() => setShowConfig((value) => !value)}
                disabled={configLoading}
                className="ui-btn-secondary px-3 py-2 text-sm"
              >
                {showConfig ? 'Hide Config' : 'Config'}
              </button>
            </div>
          </div>
          {configError && !showConfig && (
            <p className="mt-3 rounded-lg border border-rose-300/30 bg-rose-300/10 px-3 py-2 text-sm text-rose-200">{configError}</p>
          )}
        </header>

        <div className="grid flex-1 gap-4">
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

          {historyLoading && (
            <div className="ui-panel-soft animate-riseSoft px-3 py-2 text-sm text-app-muted">
              Loading session history...
            </div>
          )}

          <section className="min-h-[360px]">
            <MessageList
              messages={messages}
              loading={loading}
              onAnswerQuestion={answerQuestion}
              onCancelQuestion={cancelQuestion}
            />
          </section>

          {chatError && (
            <div className="animate-rise rounded-xl border border-rose-300/40 bg-rose-300/10 px-3 py-2 text-sm text-rose-200">
              {chatError}
            </div>
          )}

          <ChatInput
            loading={loading}
            disabled={inputDisabled}
            awaitingQuestion={hasPendingQuestion}
            onSend={handleSendChatMessage}
          />
        </div>
      </div>
    </main>
  );
};

export default HomePage;
