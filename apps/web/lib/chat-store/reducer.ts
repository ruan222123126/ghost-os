import { buildErrorMessage } from '@/lib/chatMessages';
import {
  clearPendingQuestionState,
  clearStreamingAssistantState,
  clearStreamingThinkingState,
  clearStreamingToolState,
  type PendingQuestionState,
  type StreamingAssistantState,
  type StreamingThinkingState,
  type StreamingToolTableState,
} from '@/lib/chat-stream/streamState';
import type { ActiveAgentRun } from '@/lib/chat-stream/types';
import type { ChatRuntimeAction } from '@/lib/chatRuntime/actions';
import type { ChatMessage, SessionTurnDraft } from '@/lib/types';
import { buildDraftHydratedState } from './draftHydration';
import { applyRuntimeActionsToState } from './runtimeReducer';

export interface ChatStateStore {
  committedMessages: ChatMessage[];
  streamingAssistantState: StreamingAssistantState;
  streamingThinkingState: StreamingThinkingState;
  streamingItemOrder: string[];
  streamingToolState: StreamingToolTableState;
  pendingQuestionState: PendingQuestionState;
  loading: boolean;
  historySyncing: boolean;
  historyLoading: boolean;
  loadingOlderHistory: boolean;
  chatError: string;
  activeRun: ActiveAgentRun | null;
  stopPending: boolean;
  hasOlderHistory: boolean;
  nextHistoryBefore: number | null;
}

type ScalarField =
  | 'loading'
  | 'historySyncing'
  | 'historyLoading'
  | 'loadingOlderHistory'
  | 'chatError'
  | 'activeRun'
  | 'stopPending'
  | 'hasOlderHistory'
  | 'nextHistoryBefore';

interface SetScalarAction {
  type: 'set_scalar';
  key: ScalarField;
  value: ChatStateStore[ScalarField];
}

interface SetCommittedMessagesAction {
  type: 'set_committed_messages';
  updater: ChatMessageUpdater;
}

interface ApplyRuntimeActionsAction {
  type: 'apply_runtime_actions';
  actions: ChatRuntimeAction[];
}

interface ReplaceWithErrorMessageAction {
  type: 'replace_with_error_message';
  messageText: string;
}

interface AppendErrorMessageAction {
  type: 'append_error_message';
  messageText: string;
}

interface ClearMessagesAction {
  type: 'clear_messages';
}

interface HydrateTurnDraftAction {
  type: 'hydrate_turn_draft';
  sessionId: string;
  draft: SessionTurnDraft | null | undefined;
}

export type ChatStateAction =
  | SetScalarAction
  | SetCommittedMessagesAction
  | ApplyRuntimeActionsAction
  | ReplaceWithErrorMessageAction
  | AppendErrorMessageAction
  | ClearMessagesAction
  | HydrateTurnDraftAction;

export type ChatMessageUpdater = ChatMessage[] | ((messages: ChatMessage[]) => ChatMessage[]);

export function createInitialChatState(): ChatStateStore {
  return {
    committedMessages: [],
    streamingAssistantState: clearStreamingAssistantState(),
    streamingThinkingState: clearStreamingThinkingState(),
    streamingItemOrder: [],
    streamingToolState: clearStreamingToolState(),
    pendingQuestionState: clearPendingQuestionState(),
    loading: false,
    historySyncing: false,
    historyLoading: false,
    loadingOlderHistory: false,
    chatError: '',
    activeRun: null,
    stopPending: false,
    hasOlderHistory: false,
    nextHistoryBefore: null,
  };
}

export function chatStateReducer(state: ChatStateStore, action: ChatStateAction): ChatStateStore {
  switch (action.type) {
    case 'set_scalar':
      return { ...state, [action.key]: action.value };
    case 'set_committed_messages':
      return applySetCommittedMessages(state, action.updater);
    case 'apply_runtime_actions':
      return applyRuntimeActionsToState(state, action.actions);
    case 'replace_with_error_message':
      return replaceWithErrorMessageState(state, action.messageText);
    case 'append_error_message':
      return appendErrorMessageState(state, action.messageText);
    case 'clear_messages':
      return clearMessagesState(state);
    case 'hydrate_turn_draft':
      return hydrateTurnDraftState(state, action.sessionId, action.draft);
    default:
      return state;
  }
}

function applySetCommittedMessages(
  state: ChatStateStore,
  updater: ChatMessageUpdater,
): ChatStateStore {
  const nextCommitted = typeof updater === 'function'
    ? updater(state.committedMessages)
    : updater;
  return { ...state, committedMessages: nextCommitted };
}

function replaceWithErrorMessageState(state: ChatStateStore, messageText: string): ChatStateStore {
  return {
    ...state,
    committedMessages: [buildErrorMessage(messageText)],
    streamingAssistantState: clearStreamingAssistantState(),
    streamingThinkingState: clearStreamingThinkingState(),
    streamingItemOrder: [],
    streamingToolState: clearStreamingToolState(),
    pendingQuestionState: clearPendingQuestionState(),
  };
}

function appendErrorMessageState(state: ChatStateStore, messageText: string): ChatStateStore {
  return {
    ...state,
    chatError: messageText,
    committedMessages: [...state.committedMessages, buildErrorMessage(messageText)],
  };
}

function clearMessagesState(state: ChatStateStore): ChatStateStore {
  return {
    ...state,
    chatError: '',
    committedMessages: [],
    streamingAssistantState: clearStreamingAssistantState(),
    streamingThinkingState: clearStreamingThinkingState(),
    streamingItemOrder: [],
    streamingToolState: clearStreamingToolState(),
    pendingQuestionState: clearPendingQuestionState(),
    loadingOlderHistory: false,
    historySyncing: false,
    hasOlderHistory: false,
    nextHistoryBefore: null,
  };
}

function hydrateTurnDraftState(
  state: ChatStateStore,
  sessionId: string,
  draft: SessionTurnDraft | null | undefined,
): ChatStateStore {
  const hydrated = buildDraftHydratedState(draft, sessionId.trim(), state.committedMessages);
  return {
    ...state,
    pendingQuestionState: hydrated.pendingQuestionState,
    streamingAssistantState: hydrated.streamingAssistantState,
    streamingThinkingState: hydrated.streamingThinkingState,
    streamingItemOrder: hydrated.streamingItemOrder,
    streamingToolState: hydrated.streamingToolState,
  };
}
