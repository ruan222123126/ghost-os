import type { useBridgeChat } from '@/hooks/chat/useBridgeChat';
import type { HomePageController } from '@/hooks/useHomePageController';
import type { useBridgeConfig } from '@/hooks/useBridgeConfig';
import type { useSessions } from '@/hooks/useSessions';
import { shouldShowHomeEmptyState } from '@/lib/chat-view/homeEmptyState';

type SessionsController = ReturnType<typeof useSessions>;
type ChatController = ReturnType<typeof useBridgeChat>;
type ConfigController = ReturnType<typeof useBridgeConfig>;

interface DerivedHomeStateOptions {
  chat: ChatController;
  config: ConfigController;
  sessions: SessionsController;
  showConfig: boolean;
}

export function buildDerivedHomeState(
  options: DerivedHomeStateOptions,
): Pick<HomePageController, 'inputDisabled' | 'showEmptyHomeState' | 'topStatusVisible'> {
  const { chat, config, sessions, showConfig } = options;
  const inputDisabled = config.configLoading || chat.historyLoading || !config.config || chat.hasPendingQuestion;
  const topStatusVisible = config.configLoading || (Boolean(config.configError) && !showConfig);
  const showEmptyHomeState = shouldShowHomeEmptyState({
    currentSessionId: sessions.currentSessionId,
    chat,
    showSystemPromptMessages: config.config?.session_system_prompt_visible_enabled ?? true,
  });

  return {
    inputDisabled,
    showEmptyHomeState,
    topStatusVisible,
  };
}

export function sessionControllerState(
  sessions: SessionsController,
  chat: ChatController,
): Pick<
  HomePageController,
  'backgroundCompletedSessionIds' | 'currentSessionId' | 'sessions' | 'sessionsError' | 'sessionsLoading'
> {
  return {
    sessions: sessions.sessions,
    currentSessionId: sessions.currentSessionId,
    sessionsLoading: sessions.loading,
    sessionsError: sessions.error,
    backgroundCompletedSessionIds: chat.backgroundCompletedSessionIds,
  };
}

export function chatControllerState(chat: ChatController): Pick<
  HomePageController,
  | 'activeStreamingThinkingId'
  | 'canStop'
  | 'chatError'
  | 'committedMessages'
  | 'hasOlderHistory'
  | 'hasPendingQuestion'
  | 'historyLoading'
  | 'loading'
  | 'loadingOlderHistory'
  | 'pendingQuestions'
  | 'streamingAssistantSegments'
  | 'streamingItemOrder'
  | 'streamingThinkingSegments'
  | 'streamingTools'
> {
  return {
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
  };
}

export function configControllerState(config: ConfigController): Pick<
  HomePageController,
  | 'activeModelOption'
  | 'config'
  | 'configError'
  | 'configLoading'
  | 'modelOptions'
  | 'modelOptionsLoading'
  | 'savingConfig'
> {
  return {
    config: config.config,
    configLoading: config.configLoading,
    savingConfig: config.savingConfig,
    configError: config.configError,
    modelOptionsLoading: config.modelOptionsLoading,
    activeModelOption: config.activeModelOption,
    modelOptions: config.modelOptions,
  };
}
