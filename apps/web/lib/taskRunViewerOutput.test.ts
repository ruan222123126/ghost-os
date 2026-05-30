import { buildTaskRunCardOutput } from './taskRunViewerOutput';
import type { LiveTaskRunCard } from './taskRunViewerCards';

describe('taskRunViewerOutput', () => {
  it('falls back to summary text for completed relay cards', () => {
    const output = buildTaskRunCardOutput({
      card_id: 'card-relay',
      kind: 'relay_round',
      status: 'success',
      final_text: 'did: inspect\\nnext_step: fix config',
      source_events: [],
    }, null);

    expect(output.committedMessages).toMatchObject([
      { kind: 'assistant', content: 'did: inspect\\nnext_step: fix config' },
    ]);
    expect(output.streamingRows).toHaveLength(0);
  });

  it('projects live source events into output rows', () => {
    const card: LiveTaskRunCard = {
      card_id: 'card-live',
      kind: 'workflow_agent',
      live_source_session_id: 'session-live',
      source_events: [
        {
          id: 'event-1',
          step_id: 'turn-0001-assistant',
          trace_id: 'trace-live',
          session_id: 'session-live',
          turn: 1,
          type: 'message',
          payload: { text: 'final answer', session_id: 'session-live' },
          at: '2026-05-30T00:00:00Z',
        },
      ],
    };

    const output = buildTaskRunCardOutput(card, null);

    expect(output.committedMessages).toMatchObject([
      { kind: 'assistant', content: 'final answer' },
    ]);
  });
});
