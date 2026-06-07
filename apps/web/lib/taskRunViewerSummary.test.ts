import { buildRelayCardSummary } from './taskRunViewerSummary';
import type { LiveTaskRunCard } from './taskRunViewerCards';

describe('taskRunViewerSummary', () => {
  it('extracts relay summary fields from tool output events', () => {
    const summary = buildRelayCardSummary({
      card_id: 'relay-card',
      kind: 'relay_round',
      final_text: 'did: fallback',
      source_events: [
        buildToolEvent('tool_call_started', {
          tool: 'relay_complete',
          tool_call_id: 'call-relay-complete',
          arguments_json: JSON.stringify({
            did: 'inspected state',
            next_step: 'ship result',
            final_change_log: 'changed summary layout',
          }),
        }),
        buildToolEvent('tool_call_finished', {
          tool: 'relay_complete',
          tool_call_id: 'call-relay-complete',
          status: 'success',
          output: JSON.stringify({
            status: 'relay_completed',
            did: 'inspected state',
            remaining: 'none',
            next_step: 'ship result',
            final_change_log: 'changed summary layout',
          }),
        }),
      ],
    });

    expect(summary).toEqual({
      did: 'inspected state',
      nextStep: 'ship result',
      log: 'changed summary layout',
    });
  });

  it('parses relay summary fields from persisted card text', () => {
    const summary = buildRelayCardSummary({
      card_id: 'relay-card',
      kind: 'relay_round',
      final_text: [
        'did: checked config',
        'confirmed task path',
        'remaining: none',
        'next_step: hand over',
        'final_change_log: updated viewer summary',
      ].join('\n'),
      source_events: [],
    });

    expect(summary).toEqual({
      did: 'checked config\nconfirmed task path',
      nextStep: 'hand over',
      log: 'updated viewer summary',
    });
  });

  it('does not build relay summary for non-relay cards', () => {
    const summary = buildRelayCardSummary({
      card_id: 'workflow-card',
      kind: 'workflow_agent',
      final_text: 'did: ignored',
      source_events: [],
    });

    expect(summary).toBeNull();
  });
});

function buildToolEvent(
  type: 'tool_call_started' | 'tool_call_finished',
  payload: Record<string, unknown>,
): LiveTaskRunCard['source_events'][number] {
  return {
    id: `event-${type}`,
    step_id: `step-${type}`,
    trace_id: 'trace-relay',
    session_id: 'session-relay',
    turn: 1,
    type,
    payload,
    at: '2026-05-30T00:00:00Z',
  };
}
