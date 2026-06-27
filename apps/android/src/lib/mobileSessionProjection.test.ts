import { describe, expect, it } from "vitest";
import { createAssistantConversationMessage, sessionDetailToConversationMessages } from "./mobileSessionProjection";
import type { AgentPayload, SessionDetail } from "../mobileTypes";

describe("mobileSessionProjection", () => {
  it("keeps assistant text and tool parts in bridge history order", () => {
    const messages = sessionDetailToConversationMessages(toolTagSessionDetail("session-1"));

    expect(messages).toEqual([
      expect.objectContaining({ role: "user", text: "画一只猫" }),
      expect.objectContaining({
        role: "assistant",
        text: "先开始绘图。图片已经生成。",
        parts: [
          {
            id: "session-1:1:text:0",
            kind: "text",
            text: "先开始绘图。",
          },
          {
            id: "session-1:1:tag-tool:0",
            kind: "tool",
            tool: {
              id: "session-1:1:tag-tool:0",
              input: "{\"prompt\":\"cat\"}",
              output: "Generated 1 image(s).",
              status: "success",
              toolCallId: "call-draw",
              toolName: "screen_action",
              traceId: "trace-draw",
            },
          },
          {
            id: "session-1:1:text:1",
            kind: "text",
            text: "图片已经生成。",
          },
        ],
      }),
    ]);
  });

  it("preserves ordered parts when committing assistant replies locally", () => {
    const reply: AgentPayload = {
      message: "先开始绘图。图片已经生成。",
      parts: [
        {
          id: "text:1",
          kind: "text",
          text: "先开始绘图。",
        },
        {
          id: "tool-draw",
          kind: "tool",
          tool: {
            id: "tool-draw",
            input: "{\"prompt\":\"cat\"}",
            output: "Generated 1 image(s).",
            status: "success",
            toolCallId: "call-draw",
            toolName: "screen_action",
          },
        },
        {
          id: "text:2",
          kind: "text",
          text: "图片已经生成。",
        },
      ],
      session_ended: false,
      session_id: "session-1",
    };

    expect(createAssistantConversationMessage(reply, "session-1")).toMatchObject({
      role: "assistant",
      text: "先开始绘图。图片已经生成。",
      parts: reply.parts,
    });
  });
});

function toolTagSessionDetail(id: string): SessionDetail {
  return {
    created_at: "2026-01-01T00:00:00.000Z",
    id,
    message_count: 3,
    messages: [
      {
        index: 0,
        role: "user",
        text: "画一只猫",
      },
      {
        index: 1,
        role: "assistant",
        text: "先开始绘图。<t:1>{\"prompt\":\"cat\"}</t>图片已经生成。",
      },
      {
        index: 2,
        role: "tool",
        text: "Generated 1 image(s).",
        tool_call_id: "call-draw",
        tool_result: {
          output: "Generated 1 image(s).",
          status: "success",
          tool: "screen_action",
          trace_id: "trace-draw",
        },
      },
    ],
    page: {
      has_more_before: false,
      limit: 100,
    },
    title: "Bridge session-1",
    token_count: 10,
    updated_at: "2026-01-02T00:00:00.000Z",
  };
}
