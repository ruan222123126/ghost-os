import {
  applyCardEvent,
  applyFinishedCard,
  applyStartedCard,
  hydrateLiveTaskRunCards,
  isStickyTerminalCard,
  latestActiveCard,
  mergeLiveTaskRunCards,
} from './taskRunViewerCards';

describe('taskRunViewerCards', () => {
  it('keeps cards in start-time order and tracks the latest active card', () => {
    let cards = hydrateLiveTaskRunCards([
      {
        card_id: 'card-2',
        kind: 'workflow_agent',
        started_at: '2026-05-30T00:00:02Z',
      },
      {
        card_id: 'card-1',
        kind: 'workflow_agent',
        started_at: '2026-05-30T00:00:01Z',
      },
    ]);

    cards = applyStartedCard(cards, {
      card_id: 'card-3',
      kind: 'workflow_llm',
      started_at: '2026-05-30T00:00:03Z',
    });
    cards = applyCardEvent(cards, 'card-3', {
      id: 'evt-1',
      step_id: 'turn-1-assistant',
      trace_id: 'trace-1',
      session_id: 'session-1',
      turn: 1,
      type: 'message',
      payload: { text: 'hello', session_id: 'session-1' },
      at: '2026-05-30T00:00:03Z',
    }, 'session-1');
    cards = applyFinishedCard(cards, {
      card_id: 'card-3',
      status: 'success',
      finished_at: '2026-05-30T00:00:04Z',
      preview: 'hello',
      error: undefined,
      live_source_session_id: 'session-1',
    });

    expect(cards.map((card) => card.card_id)).toEqual(['card-1', 'card-2', 'card-3']);
    expect(cards[2].source_events).toHaveLength(1);
    expect(cards[2].live_source_session_id).toBe('session-1');
    expect(latestActiveCard(cards)?.card_id).toBe('card-2');
  });

  it('treats awaiting_human and error cards as sticky terminals', () => {
    expect(isStickyTerminalCard({
      card_id: 'card-awaiting',
      kind: 'agent_task',
      status: 'awaiting_human',
      source_events: [],
    })).toBe(true);
    expect(isStickyTerminalCard({
      card_id: 'card-running',
      kind: 'agent_task',
      status: 'running',
      source_events: [],
    })).toBe(false);
  });

  it('preserves live events when refreshed run cards arrive', () => {
    const current = applyCardEvent(hydrateLiveTaskRunCards([{
      card_id: 'card-1',
      kind: 'workflow_agent',
      started_at: '2026-05-30T00:00:01Z',
    }]), 'card-1', {
      id: 'evt-1',
      step_id: 'turn-1-assistant',
      trace_id: 'trace-1',
      session_id: 'session-live',
      turn: 1,
      type: 'completion_delta',
      payload: { kind: 'text', text: 'hello' },
      at: '2026-05-30T00:00:02Z',
    }, 'session-live');

    const next = mergeLiveTaskRunCards(current, [{
      card_id: 'card-1',
      kind: 'workflow_agent',
      source_session_id: 'session-persisted',
      started_at: '2026-05-30T00:00:01Z',
      status: 'running',
      source_events: [
        {
          id: 'evt-1',
          step_id: 'turn-1-assistant',
          trace_id: 'trace-1',
          session_id: 'session-live',
          turn: 1,
          type: 'completion_delta',
          payload: { kind: 'text', text: 'hello' },
          at: '2026-05-30T00:00:02Z',
        },
      ],
    }]);

    expect(next[0].source_events).toHaveLength(1);
    expect(next[0].source_session_id).toBe('session-persisted');
    expect(next[0].live_source_session_id).toBe('session-persisted');
  });

  it('derives source session id from persisted events when card field is empty', () => {
    const cards = hydrateLiveTaskRunCards([
      {
        card_id: 'card-1',
        kind: 'workflow_agent',
        started_at: '2026-05-30T00:00:01Z',
        source_events: [
          {
            id: 'evt-1',
            step_id: 'turn-1-assistant',
            trace_id: 'trace-1',
            session_id: 'session-from-events',
            turn: 1,
            type: 'run_started',
            payload: { session_id: 'session-from-events' },
            at: '2026-05-30T00:00:01Z',
          },
        ],
      },
    ]);

    expect(cards[0].live_source_session_id).toBe('session-from-events');
  });
});
