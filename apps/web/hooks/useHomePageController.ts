'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { getSession } from '@/lib/api/sessions/api';
import { useBridgeChat } from '@/hooks/chat/useBridgeChat';
import { useBridgeConfig } from '@/hooks/useBridgeConfig';
import { useSessions } from '@/hooks/useSessions';
import {
  buildDerivedHomeState,
  chatControllerState,
  configControllerState,
  sessionControllerState,
} from '@/hooks/homePageControllerState';
import { CODEX_MODEL_IDS, DEFAULT_CODEX_MODEL, normalizeCodexModel } from '@/lib/codexModels';
import { ignorePromise } from '@/lib/errors';
import { parseSettingsQuery, stripSettingsQuery, type SettingsQueryTab } from '@/lib/settingsQuery';
import type {
  AgentModeSelection,
  ChatSendInput,
  ProviderModelOption,
  SessionRuntimeSelection,
  WorkflowTaskPayload,
} from '@/lib/types';

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
  agentMode: AgentModeSelection;
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
  setAgentMode: (mode: AgentModeSelection) => void;
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

type SessionRuntimeSelectionApplier = (selection: SessionRuntimeSelection | null | undefined) => Promise<void>;
type ComposerModelState = Pick<HomePageController, 'activeModelOption' | 'modelOptions'>;

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
  const [agentMode, setAgentMode] = useState<AgentModeSelection>(null);
  const [codexModel, setCodexModel] = useState(DEFAULT_CODEX_MODEL);
  const chat = useBridgeChat({
    currentSessionId: sessions.currentSessionId,
    externalCodexPermissionMode: config.config?.external_codex_permission_mode,
    externalProjectRoot: config.config?.project_root,
    onSessionResolved: sessions.setCurrentSessionId,
  });
  const selectActiveModel = useComposerModelSelection(agentMode, config.selectActiveModel, setCodexModel);
  const applySessionRuntimeSelection = useSessionRuntimeSelectionApplier(config.selectActiveModel, setAgentMode, setCodexModel);
  const actions = useHomePageActions(sessions, chat, applySessionRuntimeSelection);
  const derived = buildDerivedHomeState({
    chat,
    config,
    sessions,
    showConfig: settings.showConfig,
  });
  const modelState = useMemo(() => {
    return buildComposerModelState(agentMode, config.activeModelOption, config.modelOptions, codexModel);
  }, [agentMode, codexModel, config.activeModelOption, config.modelOptions]);

  return {
    ...sessionControllerState(sessions, chat),
    ...chatControllerState(chat),
    ...configControllerState(config),
    ...modelState,
    ...derived,
    showConfig: settings.showConfig,
    settingsTabFromQuery: settings.settingsTabFromQuery,
    answerQuestion: chat.answerQuestion,
    cancelQuestion: chat.cancelQuestion,
    loadOlderHistory: chat.loadOlderHistory,
    stopCurrentRun: chat.stopCurrentRun,
    saveConfig: config.saveConfig,
    selectActiveModel,
    agentMode,
    setAgentMode,
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
  applySessionRuntimeSelection: SessionRuntimeSelectionApplier,
): HomePageActions {
  const sendMessage = useCallback(async (input: ChatSendInput) => {
    await chat.sendChatMessage(input);
    await sessions.loadSessions({ silent: true });
  }, [chat, sessions]);

  const selectSession = useCallback((id: string) => {
    sessions.setCurrentSessionId(id);
    chat.clearBackgroundCompletion(id);
    const detail = chat.shouldLoadSessionHistory(id)
      ? chat.loadSessionHistory(id)
      : loadSessionRuntimeSelectionDetail(id);
    ignorePromise(detail.then((sessionDetail) => {
      return applySessionRuntimeSelection(sessionDetail?.last_runtime_selection ?? null);
    }));
  }, [applySessionRuntimeSelection, chat, sessions]);

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

function useSessionRuntimeSelectionApplier(
  selectActiveModel: HomePageController['selectActiveModel'],
  setAgentMode: (mode: AgentModeSelection) => void,
  setCodexModel: (model: string) => void,
): SessionRuntimeSelectionApplier {
  return useCallback(async (selection) => {
    if (!selection) {
      return;
    }

    setAgentMode(agentModeForRuntimeSelection(selection));
    if (selection.runtime === 'codex') {
      setCodexModel(normalizeCodexModel(selection.model));
      return;
    }

    const option = providerModelOptionForRuntimeSelection(selection);
    if (!option) {
      return;
    }
    await selectActiveModel(option);
  }, [selectActiveModel, setAgentMode, setCodexModel]);
}

function useComposerModelSelection(
  agentMode: AgentModeSelection,
  selectActiveModel: HomePageController['selectActiveModel'],
  setCodexModel: (model: string) => void,
): HomePageController['selectActiveModel'] {
  return useCallback(async (option) => {
    if (agentMode !== null && option.providerType === 'codex') {
      setCodexModel(normalizeCodexModel(option.model));
      return true;
    }

    return selectActiveModel(option);
  }, [agentMode, selectActiveModel, setCodexModel]);
}

async function loadSessionRuntimeSelectionDetail(sessionId: string) {
  return getSession(sessionId, { limit: 1 });
}

function agentModeForRuntimeSelection(selection: SessionRuntimeSelection): AgentModeSelection {
  if (selection.runtime !== 'codex') {
    return null;
  }
  return selection.mode === 'plan' ? 'plan' : 'normal';
}

function providerModelOptionForRuntimeSelection(
  selection: SessionRuntimeSelection,
): ProviderModelOption | null {
  const model = selection.model?.trim();
  if (!model) {
    return null;
  }

  const providerType = selection.runtime === 'codex'
    ? 'codex'
    : selection.provider_type ?? 'custom';
  const providerName = selection.runtime === 'codex'
    ? selection.provider?.trim() || 'codex'
    : selection.provider?.trim() || providerType;
  if (!providerName) {
    return null;
  }
  return {
    providerName,
    providerType,
    model,
  };
}

function buildComposerModelState(
  agentMode: AgentModeSelection,
  activeModelOption: ProviderModelOption | null,
  modelOptions: ProviderModelOption[],
  codexModel: string,
): ComposerModelState {
  if (agentMode === null) {
    return {
      activeModelOption,
      modelOptions,
    };
  }

  const codexOptions = buildCodexModelOptions(modelOptions);
  return {
    activeModelOption: resolveActiveCodexModelOption(codexModel, activeModelOption, codexOptions),
    modelOptions: codexOptions,
  };
}

function buildCodexModelOptions(modelOptions: ProviderModelOption[]): ProviderModelOption[] {
  const configuredOptions = uniqueProviderModelOptions(
    modelOptions.filter((option) => option.providerType === 'codex' && option.model.trim()),
  );
  if (configuredOptions.length > 0) {
    return configuredOptions;
  }

  return CODEX_MODEL_IDS.map((model) => ({
    providerName: 'codex',
    providerType: 'codex',
    model,
  }));
}

function resolveActiveCodexModelOption(
  codexModel: string,
  activeModelOption: ProviderModelOption | null,
  codexOptions: ProviderModelOption[],
): ProviderModelOption | null {
  const activeCodexModel = activeModelOption?.providerType === 'codex'
    ? activeModelOption.model.trim()
    : normalizeCodexModel(codexModel);
  const targetModel = activeCodexModel || DEFAULT_CODEX_MODEL;

  return codexOptions.find((option) => sameModel(option.model, targetModel))
    ?? codexOptions.find((option) => sameModel(option.model, DEFAULT_CODEX_MODEL))
    ?? codexOptions[0]
    ?? null;
}

function uniqueProviderModelOptions(options: ProviderModelOption[]): ProviderModelOption[] {
  const seen = new Set<string>();
  const uniqueOptions: ProviderModelOption[] = [];

  for (const option of options) {
    const model = option.model.trim();
    const providerName = option.providerName.trim() || 'codex';
    const key = `${providerName.toLowerCase()}::${model.toLowerCase()}`;
    if (seen.has(key)) {
      continue;
    }

    seen.add(key);
    uniqueOptions.push({
      providerName,
      providerType: 'codex',
      model,
    });
  }

  return uniqueOptions;
}

function sameModel(left: string, right: string): boolean {
  return left.trim().toLowerCase() === right.trim().toLowerCase();
}
