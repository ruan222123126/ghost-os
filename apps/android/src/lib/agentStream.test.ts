// @vitest-environment jsdom
import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { streamAgentMessageHTTP, type AgentStreamEvent } from "./agentStream";

vi.mock("@tauri-apps/api/core", () => ({
  invoke: vi.fn(),
}));

vi.mock("@tauri-apps/api/event", () => ({
  listen: vi.fn(),
}));

type ChunkListener = (event: { payload: { chunk: number[]; requestId: string } }) => void;

describe("streamAgentMessageHTTP reconnect", () => {
  let chunkListener: ChunkListener | undefined;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    chunkListener = undefined;
    vi.mocked(listen).mockImplementation(async (_event, handler) => {
      chunkListener = handler as ChunkListener;
      return () => undefined;
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("reconnects the same trace without resubmitting the original message", async () => {
    const received: AgentStreamEvent[] = [];
    const reconnectAttempts: number[] = [];
    vi.mocked(invoke).mockImplementation(async (command) => {
      if (command === "bridge_agent_stream") {
        emitEvents(
          agentEvent("trace-1:000001", "run_started", { session_id: "session-1" }),
          agentEvent("trace-1:000002", "completion_delta", { kind: "text", text: "前半段" }),
        );
        throw new Error("read bridge stream failed: error decoding response body");
      }
      if (vi.mocked(invoke).mock.calls.filter(([name]) => name === "bridge_agent_stream_reconnect").length === 1) {
        throw new Error("run stream is not available");
      }
      emitEvents(
        agentEvent("trace-1:000003", "completion_delta", { kind: "text", text: "后半段" }),
        agentEvent("trace-1:000004", "done", { session_ended: false, session_id: "session-1" }),
      );
    });

    const resultPromise = streamAgentMessageHTTP({
      baseUrl: "http://127.0.0.1:8711",
      message: "只发送一次",
      onEvent: (event) => received.push(event),
      onReconnectAttempt: (attempt) => reconnectAttempts.push(attempt),
      requestId: "request-1",
      traceId: "trace-1",
    });
    await vi.runAllTimersAsync();

    await expect(resultPromise).resolves.toMatchObject({
      sessionId: "session-1",
      traceId: "trace-1",
    });
    expect(vi.mocked(invoke).mock.calls.filter(([name]) => name === "bridge_agent_stream")).toHaveLength(1);
    expect(vi.mocked(invoke).mock.calls.filter(([name]) => name === "bridge_agent_stream_reconnect")).toHaveLength(2);
    expect(vi.mocked(invoke).mock.calls[2]?.[1]).toEqual({
      request: expect.objectContaining({
        lastEventId: "trace-1:000002",
        traceId: "trace-1",
      }),
    });
    expect(reconnectAttempts).toEqual([1, 2]);
    expect(received.map((event) => event.id)).toEqual([
      "trace-1:000001",
      "trace-1:000002",
      "trace-1:000003",
      "trace-1:000004",
    ]);
  });

  it("reports failure only after all three reconnect attempts fail", async () => {
    const reconnectAttempts: number[] = [];
    vi.mocked(invoke).mockRejectedValue(new Error("read bridge stream failed: error decoding response body"));

    const resultPromise = streamAgentMessageHTTP({
      baseUrl: "http://127.0.0.1:8711",
      message: "等待最终失败",
      onEvent: vi.fn(),
      onReconnectAttempt: (attempt) => reconnectAttempts.push(attempt),
      requestId: "request-2",
      traceId: "trace-2",
    });
    const rejection = expect(resultPromise).rejects.toThrow(
      "read bridge stream failed: error decoding response body",
    );
    await vi.runAllTimersAsync();

    await rejection;
    expect(reconnectAttempts).toEqual([1, 2, 3]);
    expect(vi.mocked(invoke).mock.calls.filter(([name]) => name === "bridge_agent_stream")).toHaveLength(1);
    expect(vi.mocked(invoke).mock.calls.filter(([name]) => name === "bridge_agent_stream_reconnect")).toHaveLength(3);
  });

  function emitEvents(...events: AgentStreamEvent[]): void {
    const text = events.map((event) => `event: ${event.type}\ndata: ${JSON.stringify(event)}\n\n`).join("");
    chunkListener?.({
      payload: {
        chunk: Array.from(new TextEncoder().encode(text)),
        requestId: "request-1",
      },
    });
  }
});

function agentEvent(
  id: string,
  type: AgentStreamEvent["type"],
  payload: Record<string, unknown>,
): AgentStreamEvent {
  return {
    id,
    payload,
    session_id: "session-1",
    step_id: "",
    trace_id: "trace-1",
    turn: 0,
    type,
  };
}
