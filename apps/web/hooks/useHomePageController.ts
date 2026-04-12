'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useBridgeChat } from '@/hooks/chat/useBridgeChat';
import { useBridgeConfig } from '@/hooks/useBridgeConfig';
import { useSessions } from '@/hooks/useSessions';
import { ignorePromise } from '@/lib/errors';
import { parseSettingsQuery, stripSettingsQuery, type SettingsQueryTab } from '@/lib/settingsQuery';
import type { ChatSendInput, ProviderModelOption, WorkflowTaskPayload } from '@/lib/types';

interface HomePageController {
  sessions: ReturnType<typeof useSessions>['sessions'];
  currentSessionId: string;
  sessionsLoading: boolean;
  sessionsError: string;
  committedMessages: ReturnType<typeof useBridgeChat>['committedMessages'];
  streamingAssistantSegments: ReturnType<typeof useBridgeChat>['streamingAssistantSegments'];
  streamingItemOrder: ReturnType<typeof useBridgeChat>['streamingItemOrder'];
  streamingTools: ReturnType<typeof useBridgeChat>['streamingTools'];
  pendingQuestions: ReturnType<typeof useBridgeChat>['pendingQuestions'];
  loading: boolean;
  historySyncing: boolean;
  historyLoading: boolean;
  loadingOlderHistory: boolean;
  chatError: string;
  hasPendingQuestion: boolean;
  hasOlderHistory: boolean;
  canStop: boolean;
  config: ReturnType<typeof useBridgeConfig>['config'];
  configLoading: boolean;
  savingConfig: boolean;
  configError: string;
  modelOptionsLoading: boolean;
  activeModelOption: ProviderModelOption | null;
  modelOptions: ProviderModelOption[];
  inputDisabled: boolean;
  topStatusVisible: boolean;
  showConfig: boolean;
  settingsTabFromQuery: SettingsQueryTab | null;
  answerQuestion: ReturnType<typeof useBridgeChat>['answerQuestion'];
  cancelQuestion: ReturnType<typeof useBridgeChat>['cancelQuestion'];
  loadOlderHistory: ReturnType<typeof useBridgeChat>['loadOlderHistory'];
  stopCurrentRun: ReturnType<typeof useBridgeChat>['stopCurrentRun'];
  saveConfig: ReturnType<typeof useBridgeConfig>['saveConfig'];
  selectActiveModel: ReturnType<typeof useBridgeConfig>['selectActiveModel'];
  refreshConfig: ReturnType<typeof useBridgeConfig>['refreshConfig'];
  openConfig: () => void;
  closeConfig: () => void;
  sendMessage: (input: ChatSendInput) => Promise<void>;
  selectSession: (id: string) => void;
  deleteSession: (id: string) => Promise<void>;
  newChat: () => void;
  openWorkflowCreate: () => void;
  openWorkflowEdit: (task: WorkflowTaskPayload) => void;
}

interface SettingsQueryState {
  queryString: string;
  settingsTabFromQuery: SettingsQueryTab | null;
  setQueryString: (value: string) => void;
}

function useSettingsQueryState(): SettingsQueryState {
  const [queryString, setQueryString] = useState('');

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

  return {
    queryString,
    settingsTabFromQuery: parseSettingsQuery(queryString),
    setQueryString,
  };
}

function shouldForceOpenSettings(tab: SettingsQueryTab | null): boolean {
  return tab === 'tasks' || tab === 'skills' || tab === 'tools';
}

export function useHomePageController(): HomePageController {
  const router = useRouter();
  const { queryString, settingsTabFromQuery, setQueryString } = useSettingsQueryState();
  const [showConfig, setShowConfig] = useState(false);

  useEffect(() => {
    if (shouldForceOpenSettings(settingsTabFromQuery)) {
      setShowConfig(true);
    }
  }, [settingsTabFromQuery]);

  const sessions = useSessions();
  const chat = useBridgeChat({
    currentSessionId: sessions.currentSessionId,
    onSessionResolved: sessions.setCurrentSessionId,
  });
  const config = useBridgeConfig({ autoRefresh: !showConfig });

  const handleSendMessage = useCallback(async (input: ChatSendInput) => {
    await chat.sendChatMessage(input);
    await sessions.loadSessions();
  }, [chat, sessions]);

  const handleSelectSession = useCallback((id: string) => {
    sessions.setCurrentSessionId(id);
    ignorePromise(chat.loadSessionHistory(id));
  }, [chat, sessions]);

  const handleDeleteSession = useCallback(async (id: string) => {
    await sessions.deleteSession(id);
    if (sessions.currentSessionId === id) {
      chat.clearMessages();
    }
  }, [chat, sessions]);

  const handleCloseConfig = useCallback(() => {
    setShowConfig(false);
    if (!settingsTabFromQuery) {
      return;
    }

    const nextQuery = stripSettingsQuery(queryString);
    setQueryString(nextQuery);
    router.replace(nextQuery.length > 0 ? `/${nextQuery}` : '/');
  }, [queryString, router, settingsTabFromQuery, setQueryString]);

  const inputDisabled = config.configLoading || chat.historyLoading || !config.config || chat.hasPendingQuestion;
  const topStatusVisible = config.configLoading || (Boolean(config.configError) && !showConfig);

  return {
    sessions: sessions.sessions,
    currentSessionId: sessions.currentSessionId,
    sessionsLoading: sessions.loading,
    sessionsError: sessions.error,
    committedMessages: chat.committedMessages,
    streamingAssistantSegments: chat.streamingAssistantSegments,
    streamingItemOrder: chat.streamingItemOrder,
    streamingTools: chat.streamingTools,
    pendingQuestions: chat.pendingQuestions,
    loading: chat.loading,
    historySyncing: chat.historySyncing,
    historyLoading: chat.historyLoading,
    loadingOlderHistory: chat.loadingOlderHistory,
    chatError: chat.chatError,
    hasPendingQuestion: chat.hasPendingQuestion,
    hasOlderHistory: chat.hasOlderHistory,
    canStop: chat.canStop,
    config: config.config,
    configLoading: config.configLoading,
    savingConfig: config.savingConfig,
    configError: config.configError,
    modelOptionsLoading: config.modelOptionsLoading,
    activeModelOption: config.activeModelOption,
    modelOptions: config.modelOptions,
    inputDisabled,
    topStatusVisible,
    showConfig,
    settingsTabFromQuery,
    answerQuestion: chat.answerQuestion,
    cancelQuestion: chat.cancelQuestion,
    loadOlderHistory: chat.loadOlderHistory,
    stopCurrentRun: chat.stopCurrentRun,
    saveConfig: config.saveConfig,
    selectActiveModel: config.selectActiveModel,
    refreshConfig: config.refreshConfig,
    openConfig: () => setShowConfig(true),
    closeConfig: handleCloseConfig,
    sendMessage: handleSendMessage,
    selectSession: handleSelectSession,
    deleteSession: handleDeleteSession,
    newChat: () => {
      sessions.createNewSession();
      chat.clearMessages();
    },
    openWorkflowCreate: () => {
      setShowConfig(false);
      router.push('/workflow/new');
    },
    openWorkflowEdit: (task) => {
      setShowConfig(false);
      router.push(`/workflow/${encodeURIComponent(task.id)}`);
    },
  };
}
