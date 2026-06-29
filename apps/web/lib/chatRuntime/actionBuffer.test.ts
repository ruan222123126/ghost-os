import {
  CHAT_STREAM_ACTION_FLUSH_INTERVAL_MS,
  createChatRuntimeActionBuffer,
} from './actionBuffer';
import type { ChatRuntimeAction } from './actions';

describe('chatRuntime/actionBuffer', () => {
  it('coalesces adjacent assistant text actions before flushing', () => {
    const applied: Array<{ actions: ChatRuntimeAction[]; sessionId: string }> = [];
    const timers = createManualTimers();
    const buffer = createChatRuntimeActionBuffer(
      (sessionId, actions) => applied.push({ sessionId, actions }),
      timers,
    );

    buffer.enqueue(' session-1 ', [{ type: 'append_streaming_assistant_text', text: 'hel' }]);
    buffer.enqueue('session-1', [{ type: 'append_streaming_assistant_text', text: 'lo' }]);

    expect(applied).toEqual([]);
    timers.flushNext();

    expect(timers.lastDelayMs).toBe(CHAT_STREAM_ACTION_FLUSH_INTERVAL_MS);
    expect(applied).toEqual([{
      sessionId: 'session-1',
      actions: [{ type: 'append_streaming_assistant_text', text: 'hello' }],
    }]);
  });

  it('flushes pending text before order-sensitive actions', () => {
    const applied: Array<{ actions: ChatRuntimeAction[]; sessionId: string }> = [];
    const timers = createManualTimers();
    const buffer = createChatRuntimeActionBuffer(
      (sessionId, actions) => applied.push({ sessionId, actions }),
      timers,
    );

    buffer.enqueue('session-1', [{ type: 'append_streaming_assistant_text', text: 'before' }]);
    buffer.enqueue('session-1', [
      { type: 'mark_streaming_thinking_boundary' },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'tool-1',
          content: 'running',
          toolStatus: 'running',
          traceId: 'trace-1',
        },
      },
    ]);

    expect(applied).toEqual([
      {
        sessionId: 'session-1',
        actions: [{ type: 'append_streaming_assistant_text', text: 'before' }],
      },
      {
        sessionId: 'session-1',
        actions: [
          { type: 'mark_streaming_thinking_boundary' },
          {
            type: 'upsert_streaming_tool',
            tool: {
              id: 'tool-1',
              content: 'running',
              toolStatus: 'running',
              traceId: 'trace-1',
            },
          },
        ],
      },
    ]);
  });

  it('keeps thinking and assistant segments as separate buffered actions', () => {
    const applied: Array<{ actions: ChatRuntimeAction[]; sessionId: string }> = [];
    const timers = createManualTimers();
    const buffer = createChatRuntimeActionBuffer(
      (sessionId, actions) => applied.push({ sessionId, actions }),
      timers,
    );

    buffer.enqueue('session-1', [{ type: 'append_streaming_thinking_text', text: 'plan' }]);
    buffer.enqueue('session-1', [{ type: 'append_streaming_assistant_text', text: 'answer' }]);
    buffer.flush();

    expect(applied).toEqual([
      {
        sessionId: 'session-1',
        actions: [{ type: 'append_streaming_thinking_text', text: 'plan' }],
      },
      {
        sessionId: 'session-1',
        actions: [{ type: 'append_streaming_assistant_text', text: 'answer' }],
      },
    ]);
  });
});

function createManualTimers() {
  const queue: Array<() => void> = [];
  let lastDelayMs = 0;
  return {
    clearTimer: () => undefined,
    flushNext: () => queue.shift()?.(),
    get lastDelayMs() {
      return lastDelayMs;
    },
    scheduleFlush: (flush: () => void, delayMs: number) => {
      lastDelayMs = delayMs;
      queue.push(flush);
      return queue.length as unknown as ReturnType<typeof setTimeout>;
    },
  };
}
