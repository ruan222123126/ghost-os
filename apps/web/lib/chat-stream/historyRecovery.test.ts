import type { SessionTurnDraft } from '@/lib/types';
import {
  projectRecoveredTurnEventRunState,
  projectTurnDraftRunRecovery,
} from './historyRecovery';

describe('lib/chat-stream/historyRecovery', () => {
  it('projects draft terminal states without starting a recovered run', () => {
    expect(projectTurnDraftRunRecovery('session-1', null)).toMatchObject({
      activeRun: null,
      loading: false,
      phase: 'idle',
      stopPending: false,
    });
    expect(projectTurnDraftRunRecovery('session-1', buildDraft({ status: 'awaiting_human' }))).toMatchObject({
      activeRun: null,
      chatError: '',
      loading: false,
      phase: 'awaiting_human',
    });
    expect(projectTurnDraftRunRecovery('session-1', buildDraft({ status: 'error', error: 'failed' }))).toMatchObject({
      activeRun: null,
      chatError: 'failed',
      loading: false,
      phase: 'error',
    });
  });

  it('projects streaming drafts into recovered run identity', () => {
    expect(projectTurnDraftRunRecovery('session-1', buildDraft({
      trace_id: 'trace-1',
      status: 'streaming',
    }))).toMatchObject({
      activeRun: {
        sessionId: 'session-1',
        traceId: 'trace-1',
      },
      chatError: '',
      loading: true,
      phase: 'streaming',
    });
  });

  it('projects recovered event source types into runtime projection and sync decisions', () => {
    expect(projectRecoveredTurnEventRunState('completion_delta')).toEqual({
      historySync: 'none',
      projectRuntimeEvent: true,
    });
    expect(projectRecoveredTurnEventRunState('assistant_message')).toEqual({
      historySync: 'sync',
      projectRuntimeEvent: false,
    });
    expect(projectRecoveredTurnEventRunState('awaiting_human')).toEqual({
      historySync: 'sync',
      projectRuntimeEvent: true,
    });
    expect(projectRecoveredTurnEventRunState('error')).toEqual({
      historySync: 'sync_error',
      projectRuntimeEvent: true,
    });
  });
});

function buildDraft(overrides: Partial<SessionTurnDraft>): SessionTurnDraft {
  return {
    trace_id: 'trace-1',
    turn: 1,
    status: 'streaming',
    pending_questions: [],
    assistant_segments: [],
    thinking_segments: [],
    tools: [],
    item_order: [],
    ...overrides,
  };
}
