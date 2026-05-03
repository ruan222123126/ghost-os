import { fetchMock, installFetchMock } from '@/lib/api.test.helpers';
import { streamSessionEvents } from './events';
import type { SessionPushEvent } from '@/lib/types';

describe('lib/api/sessions/events', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('streams session push events from the session events endpoint', async () => {
    fetchMock.mockResolvedValue(new Response(createSSEStream([
      buildSSEEvent({
        id: 'session-1:000001',
        type: 'completion_delta',
        trace_id: 'trace-1',
        session_id: 'session-1',
        payload: { kind: 'thinking', thinking: 'analyzing' },
      }),
      buildSSEEvent({
        id: 'session-1:000002',
        type: 'assistant_message',
        trace_id: 'trace-1',
        session_id: 'session-1',
        payload: { message: 'done', session_ended: false },
      }),
    ]), {
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
    }));

    const events: SessionPushEvent[] = [];
    await streamSessionEvents({
      onEvent: async (event) => {
        events.push(event);
      },
      sessionId: 'session-1',
    });

    expect(fetchMock).toHaveBeenCalledWith('/api/sessions/session-1/events', expect.any(Object));
    expect(events.map((event) => event.type)).toEqual([
      'completion_delta',
      'assistant_message',
    ]);
  });
});

function createSSEStream(frames: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  return new ReadableStream<Uint8Array>({
    start(controller) {
      for (const frame of frames) {
        controller.enqueue(encoder.encode(frame));
      }
      controller.close();
    },
  });
}

function buildSSEEvent(event: Omit<SessionPushEvent, 'at'> & { at?: string }): string {
  return `id: ${event.id}\nevent: ${event.type}\ndata: ${JSON.stringify(event)}\n\n`;
}
