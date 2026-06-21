'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useBridgeChat } from '@/hooks/chat/useBridgeChat';
import { useBridgeConfig } from '@/hooks/useBridgeConfig';
import { useSessions } from '@/hooks/useSessions';
import { shouldShowHomeEmptyState } from '@/lib/chat-view/homeEmptyState';
import { ignorePromise } from '@/lib/errors';
import { parseSettingsQuery, stripSettingsQuery, type SettingsQueryTab } from '@/lib/settingsQuery';
import type { ChatSendInput, ProviderModelOption, WorkflowTaskPayload } from '@/lib/types';

interface HomePageController {
  sessions: ReturnType<typeof useSessions>['sessions'];
  currentSessionId: string;
  sessionsLoading: boolean;
  sessionsError: string;
  backgroundCompletedSessionIds: ReturnType<typeof useBridgeChat>['backgroundCompletedSessionIds'];
  committedMessages: ReturnType<typeof useBridgeChat>['committedMessages'];
  streamingAssistantSegments: ReturnType<typeof useBridgeChat>['streamingAssistantSegments'];
  streamingThinkingSegments: ReturnType<typeof useBridgeChat>['streamingThinkingSegments'];
  activeStreamingThinkingId: ReturnType<typeof useBridgeChat>['activeStreamingThinkingId'];
  streamingItemOrder: ReturnType<typeof useBridgeChat>['streamingItemOrder'];
  streamingTools: ReturnType<typeof useBridgeChat>['streamingTools'];
  pendingQuestions: ReturnType<typeof useBridgeChat>['pendingQuestions'];
  loading: boolean;
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
  showEmptyHomeState: boolean;
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
  return (
    tab === 'tasks'
    || tab === 'orchestration'
    || tab === 'skills'
    || tab === 'tools'
    || tab === 'presets'
    || tab === 'prompts_library'
    || tab === 'prompts_preview'
  );
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

  const sessions = useSessions({ autoRefresh: true });
  const chat = useBridgeChat({
    currentSessionId: sessions.currentSessionId,
    onSessionResolved: sessions.setCurrentSessionId,
  });
  const config = useBridgeConfig({ autoRefresh: !showConfig });

  const handleSendMessage = useCallback(async (input: ChatSendInput) => {
    await chat.sendChatMessage(input);
    await sessions.loadSessions({ silent: true });
  }, [chat, sessions]);

  const handleSelectSession = useCallback((id: string) => {
    sessions.setCurrentSessionId(id);
    chat.clearBackgroundCompletion(id);
    if (chat.shouldLoadSessionHistory(id)) {
      ignorePromise(chat.loadSessionHistory(id));
    }
  }, [chat, sessions]);

  const handleDeleteSession = useCallback(async (id: string) => {
    await sessions.deleteSession(id);
    chat.dropSessionState(id);
    if (sessions.currentSessionId === id) {
      chat.clearMessages('');
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
  const showEmptyHomeState = shouldShowHomeEmptyState({
    currentSessionId: sessions.currentSessionId,
    chat,
    showSystemPromptMessages: config.config?.session_system_prompt_visible_enabled ?? true,
  });

  return {
    sessions: sessions.sessions,
    currentSessionId: sessions.currentSessionId,
    sessionsLoading: sessions.loading,
    sessionsError: sessions.error,
    backgroundCompletedSessionIds: chat.backgroundCompletedSessionIds,
    committedMessages: chat.committedMessages,
    streamingAssistantSegments: chat.streamingAssistantSegments,
    streamingThinkingSegments: chat.streamingThinkingSegments,
    activeStreamingThinkingId: chat.activeStreamingThinkingId,
    streamingItemOrder: chat.streamingItemOrder,
    streamingTools: chat.streamingTools,
    pendingQuestions: chat.pendingQuestions,
    loading: chat.loading,
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
    showEmptyHomeState,
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
      chat.clearMessages('');
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
