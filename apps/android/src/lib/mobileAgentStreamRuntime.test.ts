import { describe, expect, it } from "vitest";
import type { AgentStreamEvent } from "./agentStream";
import {
  createAgentPayloadFromRuntime,
  createMobileAgentStreamRuntime,
  projectMobileAgentStreamEvent,
} from "./mobileAgentStreamRuntime";
import type { MobileAgentStreamProjector } from "./mobileAgentStreamRuntime";
import type { AgentPayload, StatusMessage } from "../mobileTypes";

describe("mobile agent stream runtime", () => {
  it("projects structured tool deltas and lifecycle events into one card", () => {
    const { runtime, statuses } = projectEvents([
      event("completion_delta", {
        kind: "tool_call_start",
        tool_call_index: 0,
        tool_name: "bash_exec",
      }),
      event("completion_delta", {
        arguments_fragment: "{\"cmd\":\"",
        kind: "tool_call_delta",
        tool_call_index: 0,
      }),
      event("completion_delta", {
        arguments_fragment: "pwd\"}",
        kind: "tool_call_delta",
        tool_call_index: 0,
      }),
      event("completion_delta", {
        kind: "tool_call_end",
        tool_call_index: 0,
      }),
      event("tool_call_started", {
        arguments_json: "{\"cmd\":\"pwd\"}",
        tool: "bash_exec",
        tool_call_id: "call-1",
      }),
      event("tool_call_finished", {
        output: "/repo",
        status: "success",
        tool: "bash_exec",
        tool_call_id: "call-1",
      }),
    ]);

    expect(createAgentPayloadFromRuntime(runtime).tools).toEqual([
      {
        id: "stream-tool:trace-1:preview:1:index:0",
        input: "{\"cmd\":\"pwd\"}",
        output: "/repo",
        status: "success",
        toolCallId: "call-1",
        toolName: "bash_exec",
        traceId: "trace-1",
      },
    ]);
    expect(statuses[statuses.length - 1]).toEqual({ tone: "loading", text: "工具已返回：bash_exec" });
  });

  it("strips tool tags from visible text and creates a pending card", () => {
    const { runtime } = projectEvents([
      event("completion_delta", {
        kind: "text",
        text: "before <t:1>{\"cmd\":\"pwd\"}</t> after",
      }),
      event("message", {
        text: "before <t:1>{\"cmd\":\"pwd\"}</t> after",
      }),
    ]);

    expect(runtime.message).toBe("before  after");
    expect(createAgentPayloadFromRuntime(runtime).tools).toEqual([
      {
        id: "stream-tag-tool:trace-1:1",
        input: "{\"cmd\":\"pwd\"}",
        status: "pending",
        toolName: "tool#1",
        traceId: "trace-1",
      },
    ]);
  });

  it("stores finished tool errors explicitly", () => {
    const { runtime } = projectEvents([
      event("tool_call_started", {
        arguments_json: "{\"cmd\":\"bad\"}",
        tool: "bash_exec",
        tool_call_id: "call-error",
      }),
      event("tool_call_finished", {
        error: "command failed",
        status: "error",
        tool: "bash_exec",
        tool_call_id: "call-error",
      }),
    ]);

    expect(createAgentPayloadFromRuntime(runtime).tools).toEqual([
      {
        id: "stream-tool:trace-1:call-error",
        error: "command failed",
        input: "{\"cmd\":\"bad\"}",
        status: "error",
        toolCallId: "call-error",
        toolName: "bash_exec",
        traceId: "trace-1",
      },
    ]);
  });

  it("captures codex approval payloads for awaiting human cards", () => {
    const { runtime, statuses } = projectEvents([
      event("awaiting_human", {
        approval: {
          id: "approval-1",
          kind: "exec",
          payload: {
            command: "go test ./...",
          },
          tool: "codex_exec",
        },
        prompt: "Approve command execution",
        question_id: "approval-1",
        tool: "codex_approval",
        tool_call_id: "approval:approval-1",
      }),
    ]);

    expect(createAgentPayloadFromRuntime(runtime).tools).toEqual([
      {
        approvalId: "approval-1",
        approvalKind: "exec",
        approvalPayload: {
          command: "go test ./...",
        },
        approvalPrompt: "Approve command execution",
        id: "stream-tool:trace-1:approval:approval-1",
        input: "Approve command execution",
        status: "pending",
        toolCallId: "approval:approval-1",
        toolName: "codex_approval",
        traceId: "trace-1",
      },
    ]);
    expect(statuses[statuses.length - 1]).toEqual({ tone: "success", text: "等待用户输入" });
  });
});

function projectEvents(events: AgentStreamEvent[]) {
  const runtime = createMobileAgentStreamRuntime("session-1");
  const replies: AgentPayload[] = [];
  const statuses: StatusMessage[] = [];
  const projector: MobileAgentStreamProjector = {
    commitReply: (nextRuntime) => {
      replies.push(createAgentPayloadFromRuntime(nextRuntime));
    },
    commitSessionId: (sessionId) => {
      runtime.sessionId = sessionId;
    },
    setStatus: (status) => {
      statuses.push(status);
    },
  };

  for (const item of events) {
    projectMobileAgentStreamEvent(item, runtime, projector);
  }
  return { replies, runtime, statuses };
}

function event(type: AgentStreamEvent["type"], payload: Record<string, unknown>): AgentStreamEvent {
  return {
    id: `${type}-event`,
    payload,
    session_id: "session-1",
    step_id: `${type}-step`,
    trace_id: "trace-1",
    turn: 1,
    type,
  };
}
