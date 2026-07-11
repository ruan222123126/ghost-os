import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { getFullSession } from '@/lib/api/sessions/api';
import { streamSessionEvents } from '@/lib/api/sessions/events';
import type { SessionDetail, SessionPushEvent, TaskRunCard, TaskRunLog } from '@/lib/types';
import { useLiveRunViewer } from './useLiveRunViewer';

jest.mock('@/lib/api/sessions/api', () => ({
  getFullSession: jest.fn(),
}));

jest.mock('@/lib/api/sessions/events', () => ({
  streamSessionEvents: jest.fn(),
}));

const mockedGetFullSession = getFullSession as jest.MockedFunction<typeof getFullSession>;
const mockedStreamSessionEvents = streamSessionEvents as jest.MockedFunction<typeof streamSessionEvents>;

describe('hooks/config/useLiveRunViewer', () => {
  beforeEach(() => {
    jest.resetAllMocks();
    mockedGetFullSession.mockResolvedValue(buildSessionDetail('source-session'));
    mockedStreamSessionEvents.mockResolvedValue(undefined);
  });

  it('selects the latest active card and lets manual selection stop following latest', async () => {
    const latest = await renderViewer(buildRun({
      run_cards: [
        buildCard({ card_id: 'card-a', started_at: '2026-06-27T00:00:00Z' }),
        buildCard({ card_id: 'card-b', started_at: '2026-06-27T00:01:00Z' }),
      ],
    }));

    expect(latest.current.selectedCard?.card_id).toBe('card-b');
    expect(latest.current.followLatest).toBe(true);

    await act(async () => {
      latest.current.selectCard('card-a');
    });

    expect(latest.current.selectedCard?.card_id).toBe('card-a');
    expect(latest.current.followLatest).toBe(false);
  });

  it('loads source session details for selected cards', async () => {
    await renderViewer(buildRun({
      run_cards: [
        buildCard({
          card_id: 'card-source',
          source_session_id: 'source-session',
          status: 'success',
        }),
      ],
    }));

    expect(mockedGetFullSession).toHaveBeenCalledWith('source-session', 200);
  });

  it('applies streamed task run card events to the live card list', async () => {
    mockedStreamSessionEvents.mockImplementation(async ({ onEvent }) => {
      await onEvent(buildStartedCardEvent('card-stream'));
    });

    const latest = await renderViewer(buildRun({
      session_id_output: 'stream-session',
      run_cards: [],
    }));

    expect(mockedStreamSessionEvents).toHaveBeenCalledWith(expect.objectContaining({
      sessionId: 'stream-session',
    }));
    expect(latest.current.cards.map((card) => card.card_id)).toEqual(['card-stream']);
  });
});

async function renderViewer(run: TaskRunLog) {
  const latest: { current: ReturnType<typeof useLiveRunViewer> } = {
    current: null as unknown as ReturnType<typeof useLiveRunViewer>,
  };

  await act(async () => {
    TestRenderer.create(
      React.createElement(LiveRunViewerProbe, {
        run,
        onRender: (state) => {
          latest.current = state;
        },
      }),
    );
    await flushPromises();
  });

  return latest;
}

function LiveRunViewerProbe(props: {
  run: TaskRunLog;
  onRender: (state: ReturnType<typeof useLiveRunViewer>) => void;
}) {
  const state = useLiveRunViewer({ run: props.run });
  props.onRender(state);
  return null;
}

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
}

function buildRun(overrides: Partial<TaskRunLog> = {}): TaskRunLog {
  return {
    task_id: 'task-1',
    run_id: 'run-1',
    trace_id: 'trace-1',
    scheduled_at: '2026-06-27T00:00:00Z',
    status: 'running',
    ...overrides,
  };
}

function buildCard(overrides: Partial<TaskRunCard> = {}): TaskRunCard {
  return {
    card_id: 'card-1',
    run_id: 'run-1',
    kind: 'agent',
    started_at: '2026-06-27T00:00:00Z',
    status: 'running',
    ...overrides,
  };
}

function buildStartedCardEvent(cardID: string): SessionPushEvent {
  return {
    id: `event-${cardID}`,
    type: 'task_run_card_started',
    trace_id: 'trace-1',
    session_id: 'stream-session',
    at: '2026-06-27T00:02:00Z',
    payload: {
      card_id: cardID,
      run_id: 'run-1',
      kind: 'agent',
      started_at: '2026-06-27T00:02:00Z',
    },
  };
}

function buildSessionDetail(id: string): SessionDetail {
  return {
    id,
    title: id,
    messages: [],
    created_at: '2026-06-27T00:00:00Z',
    updated_at: '2026-06-27T00:00:00Z',
    message_count: 0,
    page: {
      limit: 200,
      has_more_before: false,
      next_before: null,
      start_index: null,
      end_index: null,
    },
    token_count: 0,
    turn_draft: null,
  };
}
