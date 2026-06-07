import type { AgentStreamEvent } from '@/lib/types';
import { resolveEventSession, resolveStreamSession } from './runSession';
import { resolveEventSessionId } from './sessionEvent';

describe('lib/chat-stream/runSession', () => {
  it('resolves session id from top-level event or event payload', () => {
    expect(resolveEventSessionId(buildEvent('run_started', { session_id: 'session-payload' }, ''))).toBe(
      'session-payload',
    );
    expect(resolveEventSessionId(buildEvent('message', { text: 'done', session_id: 'session-message' }, ''))).toBe(
      'session-message',
    );
    expect(resolveEventSessionId(buildEvent('done', { session_id: 'session-top' }, 'session-top'))).toBe(
      'session-top',
    );
  });

  it('does not resolve unchanged or missing event session ids', () => {
    expect(resolveEventSession({
      activeRun: null,
      currentSessionId: 'session-1',
      event: buildEvent('completion_delta', { kind: 'text', text: 'hello' }, ''),
      runtimeSessionId: 'session-1',
    })).toBeNull();
    expect(resolveEventSession({
      activeRun: null,
      currentSessionId: 'session-1',
      event: buildEvent('done', { session_id: 'session-1' }, ''),
      runtimeSessionId: 'session-1',
    })).toBeNull();
  });

  it('updates active run and notifies when stream resolves a new session', () => {
    expect(resolveStreamSession({
      activeRun: {
        sessionId: '',
        traceId: 'trace-1',
      },
      currentSessionId: '',
      sessionId: 'session-1',
    })).toEqual({
      nextActiveRun: {
        sessionId: 'session-1',
        traceId: 'trace-1',
      },
      notifySessionResolved: true,
      sessionId: 'session-1',
    });
  });
});

function buildEvent(
  type: AgentStreamEvent['type'],
  payload: Record<string, unknown>,
  sessionId: string,
): AgentStreamEvent {
  return {
    id: `${type}-1`,
    step_id: '',
    trace_id: 'trace-1',
    session_id: sessionId || undefined,
    turn: 1,
    type,
    payload,
  };
}
