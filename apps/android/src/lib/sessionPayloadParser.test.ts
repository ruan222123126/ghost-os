import { describe, expect, it } from "vitest";
import { parseSessionDetail, parseSessionMetadataList } from "./sessionPayloadParser";

describe("session payload parser", () => {
  it("parses valid session metadata list", () => {
    const sessions = parseSessionMetadataList([sessionMetadata("session-1")]);

    expect(sessions).toEqual([sessionMetadata("session-1")]);
  });

  it("parses valid session detail", () => {
    const detail = parseSessionDetail({
      ...sessionMetadata("session-1"),
      messages: [
        {
          index: 0,
          role: "user",
          text: "hello",
        },
        {
          content: [{ type: "text", text: "done" }],
          index: 1,
          role: "assistant",
          thinking: "checking",
          tool_calls: [
            {
              arguments: { cmd: "pwd" },
              id: "call-1",
              name: "bash_exec",
            },
          ],
        },
        {
          index: 2,
          role: "tool",
          text: "/repo",
          tool_call_id: "call-1",
          tool_result: {
            output: "/repo",
            status: "success",
            tool: "bash_exec",
            trace_id: "trace-1",
          },
        },
      ],
      page: {
        before: null,
        has_more_before: false,
        limit: 100,
        next_before: null,
      },
      turn_draft: {
        trace_id: "trace-draft",
        turn: 2,
        status: "awaiting_human",
        pending_questions: [
          {
            question_id: "approval-1",
            prompt: "Approve command?",
            selection_mode: "single",
            options: [{ label: "Approve", allow_custom: false }],
          },
        ],
        assistant_segments: [{ id: "stream-segment:assistant:1", content: "partial" }],
        thinking_segments: [{ id: "stream-segment:thinking:1", content: "thinking" }],
        tools: [
          {
            id: "stream-tool:trace-draft:call-1",
            content: "/repo",
            tool_input: "{\"cmd\":\"pwd\"}",
            tool_name: "bash_exec",
            tool_status: "success",
            tool_call_id: "call-1",
            trace_id: "trace-draft",
          },
        ],
        item_order: [
          "assistant:stream-segment:assistant:1",
          "tool:stream-tool:trace-draft:call-1",
          "question:approval-1",
        ],
      },
      last_runtime_selection: {
        runtime: "ghost",
        provider: "OpenAI Main",
        provider_type: "openai",
        model: "gpt-5.4",
        mode: "plan",
      },
    });

    expect(detail.id).toBe("session-1");
    expect(detail.messages).toHaveLength(3);
    expect(detail.messages[1]?.tool_calls).toEqual([
      { arguments: { cmd: "pwd" }, id: "call-1", name: "bash_exec" },
    ]);
    expect(detail.messages[2]?.tool_call_id).toBe("call-1");
    expect(detail.messages[2]?.tool_result).toEqual({
      output: "/repo",
      status: "success",
      tool: "bash_exec",
      trace_id: "trace-1",
    });
    expect(detail.last_runtime_selection).toEqual({
      runtime: "ghost",
      provider: "OpenAI Main",
      provider_type: "openai",
      model: "gpt-5.4",
      mode: "plan",
    });
    expect(detail.turn_draft).toEqual({
      trace_id: "trace-draft",
      turn: 2,
      status: "awaiting_human",
      pending_questions: [
        {
          question_id: "approval-1",
          prompt: "Approve command?",
          selection_mode: "single",
          options: [{ label: "Approve", allow_custom: false }],
        },
      ],
      assistant_segments: [{ id: "stream-segment:assistant:1", content: "partial" }],
      thinking_segments: [{ id: "stream-segment:thinking:1", content: "thinking" }],
      tools: [
        {
          id: "stream-tool:trace-draft:call-1",
          content: "/repo",
          tool_input: "{\"cmd\":\"pwd\"}",
          tool_name: "bash_exec",
          tool_status: "success",
          tool_call_id: "call-1",
          trace_id: "trace-draft",
        },
      ],
      item_order: [
        "assistant:stream-segment:assistant:1",
        "tool:stream-tool:trace-draft:call-1",
        "question:approval-1",
      ],
    });
    expect(detail.page.limit).toBe(100);
  });

  it("rejects metadata entries missing required fields", () => {
    expect(() => parseSessionMetadataList([{ id: "session-1" }])).toThrow(
      "SESSIONS_LIST payload[0].title must be a string",
    );
  });

  it("rejects messages with invalid roles", () => {
    expect(() =>
      parseSessionDetail({
        ...sessionMetadata("session-1"),
        messages: [{ index: 0, role: "owner", text: "bad" }],
        page: { has_more_before: false, limit: 100 },
      }),
    ).toThrow("SESSION_GET payload.messages[0].role must be one of");
  });

  it("rejects detail payloads missing page fields", () => {
    expect(() =>
      parseSessionDetail({
        ...sessionMetadata("session-1"),
        messages: [],
        page: { limit: 100 },
      }),
    ).toThrow("SESSION_GET payload.page.has_more_before must be a boolean");
  });

  it("rejects invalid session tool call fields", () => {
    expect(() =>
      parseSessionDetail({
        ...sessionMetadata("session-1"),
        messages: [
          {
            index: 0,
            role: "assistant",
            tool_calls: [{ arguments: "bad", id: "call-1", name: "bash_exec" }],
          },
        ],
        page: { has_more_before: false, limit: 100 },
      }),
    ).toThrow("SESSION_GET payload.messages[0].tool_calls[0].arguments must be an object");
  });

  it("rejects invalid session tool result fields", () => {
    expect(() =>
      parseSessionDetail({
        ...sessionMetadata("session-1"),
        messages: [
          {
            index: 0,
            role: "tool",
            tool_result: { status: "done", tool: "bash_exec" },
          },
        ],
        page: { has_more_before: false, limit: 100 },
      }),
    ).toThrow("SESSION_GET payload.messages[0].tool_result.status must be one of success, error");
  });

  it("rejects errored turn drafts without an error message", () => {
    expect(() =>
      parseSessionDetail({
        ...sessionMetadata("session-1"),
        messages: [],
        page: { has_more_before: false, limit: 100 },
        turn_draft: {
          trace_id: "trace-draft",
          turn: 1,
          status: "error",
          pending_questions: [],
          assistant_segments: [],
          thinking_segments: [],
          tools: [],
          item_order: [],
        },
      }),
    ).toThrow("SESSION_GET payload.turn_draft.error must be a string when status=error");
  });
});

function sessionMetadata(id: string) {
  return {
    created_at: "2026-01-01T00:00:00.000Z",
    id,
    message_count: 1,
    title: "Session",
    token_count: 2,
    updated_at: "2026-01-02T00:00:00.000Z",
  };
}
