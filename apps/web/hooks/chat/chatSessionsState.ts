import {
  chatStateReducer,
  createInitialChatState,
  type ChatStateAction,
  type ChatStateStore,
} from '@/lib/chat-store/reducer';
import { buildChatStateView } from '@/lib/chat-store/runtimeReducer';
import type { ActiveAgentRun } from './types';

export interface ChatSessionsStateStore {
  backgroundCompletedSessionIds: string[];
  sessionsById: Record<string, ChatStateStore>;
}

export type ChatSessionsStateAction =
  | {
    type: 'apply_session_action';
    action: ChatStateAction;
    sessionId: string;
  }
  | {
    type: 'migrate_session_state';
    fromSessionId: string;
    toSessionId: string;
  }
  | {
    type: 'mark_background_completed';
    sessionId: string;
  }
  | {
    type: 'clear_background_completed';
    sessionId: string;
  }
  | {
    type: 'drop_session_state';
    sessionId: string;
  };

export interface RuntimeRefsMove {
  activeRuns: Record<string, ActiveAgentRun>;
  fromKey: string;
  historySyncCounts: Record<string, number>;
  stopPending: Record<string, boolean>;
  toKey: string;
}

export function createInitialChatSessionsState(): ChatSessionsStateStore {
  return {
    backgroundCompletedSessionIds: [],
    sessionsById: {},
  };
}

export function chatSessionsReducer(
  state: ChatSessionsStateStore,
  action: ChatSessionsStateAction,
): ChatSessionsStateStore {
  switch (action.type) {
    case 'apply_session_action':
      return applySessionActionState(state, action.sessionId, action.action);
    case 'migrate_session_state':
      return migrateSessionState(state, action.fromSessionId, action.toSessionId);
    case 'mark_background_completed':
      return markBackgroundCompletedState(state, action.sessionId);
    case 'clear_background_completed':
      return clearBackgroundCompletedState(state, action.sessionId);
    case 'drop_session_state':
      return dropSessionState(state, action.sessionId);
    default:
      return state;
  }
}

export function getSessionState(state: ChatSessionsStateStore, sessionId: string): ChatStateStore {
  return state.sessionsById[normalizeSessionId(sessionId)] ?? createInitialChatState();
}

export function shouldLoadSessionHistory(state: ChatSessionsStateStore, sessionId: string): boolean {
  const key = normalizeSessionId(sessionId);
  if (!key) {
    return false;
  }
  if (!hasSessionState(state, key)) {
    return true;
  }

  const sessionState = getSessionState(state, key);
  const view = buildChatStateView(sessionState);
  return !sessionState.loading
    && !sessionState.activeRun
    && !sessionState.historyLoading
    && view.pendingQuestions.length === 0
    && view.streamingAssistantSegments.length === 0
    && view.streamingThinkingSegments.length === 0
    && view.streamingTools.length === 0;
}

export function moveRuntimeRefs(options: RuntimeRefsMove): void {
  const { activeRuns, fromKey, historySyncCounts, stopPending, toKey } = options;

  const activeRun = activeRuns[fromKey];
  if (activeRun) {
    activeRuns[toKey] = { ...activeRun, sessionId: toKey };
    delete activeRuns[fromKey];
  }
  if (stopPending[fromKey]) {
    stopPending[toKey] = true;
    delete stopPending[fromKey];
  }
  if (historySyncCounts[fromKey]) {
    historySyncCounts[toKey] = (historySyncCounts[toKey] ?? 0) + historySyncCounts[fromKey];
    delete historySyncCounts[fromKey];
  }
}

export function resolveActiveRunSessionId(
  activeRuns: Record<string, ActiveAgentRun>,
  traceId: string,
  fallbackSessionId: string,
): string {
  const trimmedTraceId = traceId.trim();
  for (const [sessionId, run] of Object.entries(activeRuns)) {
    if (run.traceId === trimmedTraceId) {
      return normalizeSessionId(run.sessionId) || sessionId;
    }
  }
  return normalizeSessionId(fallbackSessionId);
}

export function normalizeSessionId(sessionId: string): string {
  return sessionId.trim();
}

function applySessionActionState(
  state: ChatSessionsStateStore,
  sessionId: string,
  action: ChatStateAction,
): ChatSessionsStateStore {
  const key = normalizeSessionId(sessionId);
  const previous = state.sessionsById[key] ?? createInitialChatState();
  return {
    ...state,
    sessionsById: {
      ...state.sessionsById,
      [key]: chatStateReducer(previous, action),
    },
  };
}

function migrateSessionState(
  state: ChatSessionsStateStore,
  fromSessionId: string,
  toSessionId: string,
): ChatSessionsStateStore {
  const fromKey = normalizeSessionId(fromSessionId);
  const toKey = normalizeSessionId(toSessionId);
  const source = state.sessionsById[fromKey];
  if (!source || !toKey || fromKey === toKey) {
    return state;
  }

  const { [fromKey]: _removed, ...remaining } = state.sessionsById;
  return {
    ...state,
    sessionsById: {
      ...remaining,
      [toKey]: resolveMigratedState(source, toKey),
    },
  };
}

function markBackgroundCompletedState(
  state: ChatSessionsStateStore,
  sessionId: string,
): ChatSessionsStateStore {
  const key = normalizeSessionId(sessionId);
  if (!key || state.backgroundCompletedSessionIds.includes(key)) {
    return state;
  }
  return {
    ...state,
    backgroundCompletedSessionIds: [...state.backgroundCompletedSessionIds, key],
  };
}

function clearBackgroundCompletedState(
  state: ChatSessionsStateStore,
  sessionId: string,
): ChatSessionsStateStore {
  const key = normalizeSessionId(sessionId);
  const nextIds = state.backgroundCompletedSessionIds.filter((id) => id !== key);
  return nextIds.length === state.backgroundCompletedSessionIds.length
    ? state
    : { ...state, backgroundCompletedSessionIds: nextIds };
}

function dropSessionState(
  state: ChatSessionsStateStore,
  sessionId: string,
): ChatSessionsStateStore {
  const key = normalizeSessionId(sessionId);
  if (!hasSessionState(state, key) && !state.backgroundCompletedSessionIds.includes(key)) {
    return state;
  }
  const { [key]: _removed, ...remaining } = state.sessionsById;
  return {
    backgroundCompletedSessionIds: state.backgroundCompletedSessionIds.filter((id) => id !== key),
    sessionsById: remaining,
  };
}

function hasSessionState(state: ChatSessionsStateStore, sessionId: string): boolean {
  return Object.prototype.hasOwnProperty.call(state.sessionsById, normalizeSessionId(sessionId));
}

function resolveMigratedState(state: ChatStateStore, sessionId: string): ChatStateStore {
  if (!state.activeRun || state.activeRun.sessionId === sessionId) {
    return state;
  }
  return {
    ...state,
    activeRun: {
      ...state.activeRun,
      sessionId,
    },
  };
}
