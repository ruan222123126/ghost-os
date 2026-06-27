'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useBridgeChat } from '@/hooks/chat/useBridgeChat';
import { useBridgeConfig } from '@/hooks/useBridgeConfig';
import { useSessions } from '@/hooks/useSessions';
import {
  buildDerivedHomeState,
  chatControllerState,
  configControllerState,
  sessionControllerState,
} from '@/hooks/homePageControllerState';
import { ignorePromise } from '@/lib/errors';
import { parseSettingsQuery, stripSettingsQuery, type SettingsQueryTab } from '@/lib/settingsQuery';
import type { ChatSendInput, ProviderModelOption, WorkflowTaskPayload } from '@/lib/types';

export interface HomePageController {
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

type SessionsController = ReturnType<typeof useSessions>;
type ChatController = ReturnType<typeof useBridgeChat>;
type RouterController = ReturnType<typeof useRouter>;

interface SettingsPanelState {
  closeConfig: () => void;
  openConfig: () => void;
  openWorkflowCreate: () => void;
  openWorkflowEdit: (task: WorkflowTaskPayload) => void;
  settingsTabFromQuery: SettingsQueryTab | null;
  showConfig: boolean;
}

interface HomePageActions {
  deleteSession: HomePageController['deleteSession'];
  newChat: HomePageController['newChat'];
  selectSession: HomePageController['selectSession'];
  sendMessage: HomePageController['sendMessage'];
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
  const settings = useSettingsPanelState(router);
  const sessions = useSessions({ autoRefresh: true });
  const config = useBridgeConfig({ autoRefresh: !settings.showConfig });
  const chat = useBridgeChat({
    currentSessionId: sessions.currentSessionId,
    externalCodexPermissionMode: config.config?.external_codex_permission_mode,
    externalProjectRoot: config.config?.project_root,
    onSessionResolved: sessions.setCurrentSessionId,
  });
  const actions = useHomePageActions(sessions, chat);
  const derived = buildDerivedHomeState({
    chat,
    config,
    sessions,
    showConfig: settings.showConfig,
  });

  return {
    ...sessionControllerState(sessions, chat),
    ...chatControllerState(chat),
    ...configControllerState(config),
    ...derived,
    showConfig: settings.showConfig,
    settingsTabFromQuery: settings.settingsTabFromQuery,
    answerQuestion: chat.answerQuestion,
    cancelQuestion: chat.cancelQuestion,
    loadOlderHistory: chat.loadOlderHistory,
    stopCurrentRun: chat.stopCurrentRun,
    saveConfig: config.saveConfig,
    selectActiveModel: config.selectActiveModel,
    refreshConfig: config.refreshConfig,
    openConfig: settings.openConfig,
    closeConfig: settings.closeConfig,
    sendMessage: actions.sendMessage,
    selectSession: actions.selectSession,
    deleteSession: actions.deleteSession,
    newChat: actions.newChat,
    openWorkflowCreate: settings.openWorkflowCreate,
    openWorkflowEdit: settings.openWorkflowEdit,
  };
}

function useSettingsPanelState(router: RouterController): SettingsPanelState {
  const { queryString, settingsTabFromQuery, setQueryString } = useSettingsQueryState();
  const [showConfig, setShowConfig] = useState(false);

  useEffect(() => {
    if (shouldForceOpenSettings(settingsTabFromQuery)) {
      setShowConfig(true);
    }
  }, [settingsTabFromQuery]);

  const closeConfig = useCallback(() => {
    setShowConfig(false);
    if (!settingsTabFromQuery) {
      return;
    }

    const nextQuery = stripSettingsQuery(queryString);
    setQueryString(nextQuery);
    router.replace(nextQuery.length > 0 ? `/${nextQuery}` : '/');
  }, [queryString, router, settingsTabFromQuery, setQueryString]);

  const openWorkflowCreate = useCallback(() => {
    setShowConfig(false);
    router.push('/workflow/new');
  }, [router]);

  const openWorkflowEdit = useCallback((task: WorkflowTaskPayload) => {
    setShowConfig(false);
    router.push(`/workflow/${encodeURIComponent(task.id)}`);
  }, [router]);

  return {
    closeConfig,
    openConfig: () => setShowConfig(true),
    openWorkflowCreate,
    openWorkflowEdit,
    settingsTabFromQuery,
    showConfig,
  };
}

function useHomePageActions(
  sessions: SessionsController,
  chat: ChatController,
): HomePageActions {
  const sendMessage = useCallback(async (input: ChatSendInput) => {
    await chat.sendChatMessage(input);
    await sessions.loadSessions({ silent: true });
  }, [chat, sessions]);

  const selectSession = useCallback((id: string) => {
    sessions.setCurrentSessionId(id);
    chat.clearBackgroundCompletion(id);
    if (chat.shouldLoadSessionHistory(id)) {
      ignorePromise(chat.loadSessionHistory(id));
    }
  }, [chat, sessions]);

  const deleteSession = useCallback(async (id: string) => {
    await sessions.deleteSession(id);
    chat.dropSessionState(id);
    if (sessions.currentSessionId === id) {
      chat.clearMessages('');
    }
  }, [chat, sessions]);

  const newChat = useCallback(() => {
    sessions.createNewSession();
    chat.clearMessages('');
  }, [chat, sessions]);

  return { deleteSession, newChat, selectSession, sendMessage };
}
