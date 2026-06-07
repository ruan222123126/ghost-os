import type { AgentStreamEvent, SessionPushEvent, SessionTurnDraft } from '@/lib/types';
import type { ActiveAgentRun } from './types';

type RecoveredTurnPhase = 'idle' | 'awaiting_human' | 'error' | 'streaming';
type RecoveredTurnSourceType = SessionPushEvent['type'] | AgentStreamEvent['type'];
export type RecoveredTurnEventHistorySync = 'none' | 'sync' | 'sync_error';

export interface TurnDraftRunRecoveryProjection {
  activeRun: Pick<ActiveAgentRun, 'sessionId' | 'traceId'> | null;
  chatError?: string;
  loading: boolean;
  phase: RecoveredTurnPhase;
  stopPending: false;
}

export interface RecoveredTurnEventRunState {
  historySync: RecoveredTurnEventHistorySync;
  projectRuntimeEvent: boolean;
}

export function projectTurnDraftRunRecovery(
  sessionId: string,
  draft: SessionTurnDraft | null | undefined,
): TurnDraftRunRecoveryProjection {
  if (!draft) {
    return {
      activeRun: null,
      loading: false,
      phase: 'idle',
      stopPending: false,
    };
  }

  if (draft.status === 'awaiting_human') {
    return {
      activeRun: null,
      chatError: '',
      loading: false,
      phase: 'awaiting_human',
      stopPending: false,
    };
  }

  if (draft.status === 'error') {
    return {
      activeRun: null,
      chatError: draft.error ?? '',
      loading: false,
      phase: 'error',
      stopPending: false,
    };
  }

  return {
    activeRun: {
      sessionId,
      traceId: draft.trace_id,
    },
    chatError: '',
    loading: true,
    phase: 'streaming',
    stopPending: false,
  };
}

export function projectRecoveredTurnEventRunState(
  sourceType: RecoveredTurnSourceType,
): RecoveredTurnEventRunState {
  switch (sourceType) {
    case 'assistant_message':
    case 'message':
      return {
        historySync: 'sync',
        projectRuntimeEvent: false,
      };
    case 'awaiting_human':
      return {
        historySync: 'sync',
        projectRuntimeEvent: true,
      };
    case 'completion_delta':
    case 'tool_call_started':
    case 'tool_call_finished':
    case 'run_started':
    case 'done':
      return {
        historySync: 'none',
        projectRuntimeEvent: true,
      };
    case 'error':
      return {
        historySync: 'sync_error',
        projectRuntimeEvent: true,
      };
    default:
      return {
        historySync: 'none',
        projectRuntimeEvent: false,
      };
  }
}
