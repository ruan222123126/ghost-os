import { fetchMock, installFetchMock } from '@/lib/api.test.helpers';
import { streamExternalMessage, streamMessage } from './stream';
import type { AgentStreamEvent } from '@/lib/types';

describe('lib/api/agent/stream', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('streams SSE events and returns the final result', async () => {
    fetchMock.mockResolvedValue(new Response(createSSEStream([
      buildSSEEvent({
        id: 'trace-1:000001',
        step_id: '',
        trace_id: 'trace-1',
        session_id: 'session-1',
        turn: 0,
        type: 'run_started',
        payload: { session_id: 'session-1' },
      }),
      buildSSEEvent({
        id: 'trace-1:000002',
        step_id: 'assistant-1',
        trace_id: 'trace-1',
        session_id: 'session-1',
        turn: 1,
        type: 'completion_delta',
        payload: { kind: 'text', text: 'hel' },
      }),
      buildSSEEvent({
        id: 'trace-1:000003',
        step_id: 'assistant-1',
        trace_id: 'trace-1',
        session_id: 'session-1',
        turn: 1,
        type: 'completion_delta',
        payload: { kind: 'text', text: 'lo' },
      }),
      buildSSEEvent({
        id: 'trace-1:000004',
        step_id: 'assistant-1',
        trace_id: 'trace-1',
        session_id: 'session-1',
        turn: 1,
        type: 'message',
        payload: { text: 'hello', session_id: 'session-1' },
      }),
      buildSSEEvent({
        id: 'trace-1:000005',
        step_id: '',
        trace_id: 'trace-1',
        session_id: 'session-1',
        turn: 1,
        type: 'done',
        payload: { session_id: 'session-1', session_ended: false },
      }),
    ]), {
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
    }));

    const events: AgentStreamEvent[] = [];
    const result = await streamMessage({
      message: 'hello',
      onEvent: async (event) => {
        events.push(event);
      },
      traceId: 'trace-1',
    });

    expect(result).toEqual({
      message: 'hello',
      sessionEnded: false,
      sessionId: 'session-1',
      traceId: 'trace-1',
    });
    expect(events.map((event) => event.type)).toEqual([
      'run_started',
      'completion_delta',
      'completion_delta',
      'message',
      'done',
    ]);

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(fetchMock).toHaveBeenCalledWith('/api/agent/stream', expect.objectContaining({ method: 'POST' }));
    expect(JSON.parse(String(init.body))).toEqual({
      message: 'hello',
      trace_id: 'trace-1',
    });
  });

  it('throws the envelope error when the bridge does not return SSE', async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({
      status: 'error',
      payload: {},
      error: 'session is already running',
    }), {
      status: 409,
      headers: { 'Content-Type': 'application/json' },
    }));

    await expect(streamMessage({
      message: 'hello',
      onEvent: async () => undefined,
      traceId: 'trace-1',
    })).rejects.toThrow('session is already running');
  });

  it('includes image inputs in the streamed agent request', async () => {
    fetchMock.mockResolvedValue(new Response(createSSEStream([
      buildSSEEvent({
        id: 'trace-2:000001',
        step_id: '',
        trace_id: 'trace-2',
        session_id: 'session-2',
        turn: 0,
        type: 'run_started',
        payload: { session_id: 'session-2' },
      }),
      buildSSEEvent({
        id: 'trace-2:000002',
        step_id: 'assistant-2',
        trace_id: 'trace-2',
        session_id: 'session-2',
        turn: 1,
        type: 'message',
        payload: { text: 'ok', session_id: 'session-2' },
      }),
      buildSSEEvent({
        id: 'trace-2:000003',
        step_id: '',
        trace_id: 'trace-2',
        session_id: 'session-2',
        turn: 1,
        type: 'done',
        payload: { session_id: 'session-2', session_ended: false },
      }),
    ]), {
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
    }));

    await streamMessage({
      images: [{
        url: 'data:image/png;base64,R2hvc3Q=',
        mime_type: 'image/png',
        bytes: 5,
      }],
      message: '',
      onEvent: async () => undefined,
      traceId: 'trace-2',
    });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(JSON.parse(String(init.body))).toEqual({
      images: [{
        url: 'data:image/png;base64,R2hvc3Q=',
        mime_type: 'image/png',
        bytes: 5,
      }],
      message: '',
      trace_id: 'trace-2',
    });
  });

  it('includes the selected model in external Codex requests', async () => {
    fetchMock.mockResolvedValue(new Response(createSSEStream([
      buildSSEEvent({
        id: 'trace-codex:000001',
        step_id: '',
        trace_id: 'trace-codex',
        session_id: 'session-codex',
        turn: 0,
        type: 'done',
        payload: { session_id: 'session-codex', session_ended: false },
      }),
    ]), {
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
    }));

    await streamExternalMessage({
      message: 'ship release',
      model: ' gpt-5.5 ',
      onEvent: async () => undefined,
      permissionMode: 'safe-yolo',
      traceId: 'trace-codex',
    });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(fetchMock).toHaveBeenCalledWith('/api/external-agent/stream', expect.objectContaining({ method: 'POST' }));
    expect(JSON.parse(String(init.body))).toEqual({
      message: 'ship release',
      model: 'gpt-5.5',
      permission_mode: 'safe-yolo',
      provider: 'codex',
    });
  });
});

function createSSEStream(frames: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  return new ReadableStream<Uint8Array>({
    start(controller) {
      for (const frame of frames) {
        const midpoint = Math.ceil(frame.length / 2);
        controller.enqueue(encoder.encode(frame.slice(0, midpoint)));
        controller.enqueue(encoder.encode(frame.slice(midpoint)));
      }
      controller.close();
    },
  });
}

function buildSSEEvent(event: AgentStreamEvent): string {
  return `id: ${event.id}\nevent: ${event.type}\ndata: ${JSON.stringify(event)}\n\n`;
}
