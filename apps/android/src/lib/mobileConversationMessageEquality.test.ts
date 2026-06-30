import { describe, expect, it } from "vitest";
import type { MobileConversationMessage } from "../mobileTypes";
import { areMobileConversationMessagesEqual } from "./mobileConversationMessageEquality";

describe("mobileConversationMessageEquality", () => {
  it("treats copied messages with the same renderable content as equal", () => {
    const message = assistantMessage();

    expect(areMobileConversationMessagesEqual(message, assistantMessage())).toBe(true);
  });

  it("detects assistant text changes", () => {
    const message = assistantMessage();
    const changed = {
      ...assistantMessage(),
      parts: [{ id: "text-1", kind: "text", text: "changed" }],
      text: "changed",
    } satisfies MobileConversationMessage;

    expect(areMobileConversationMessagesEqual(message, changed)).toBe(false);
  });

  it("detects tool status changes", () => {
    const message = assistantMessage();
    const changed = {
      ...assistantMessage(),
      tools: [{
        id: "tool-1",
        input: "pwd",
        output: "/repo",
        status: "error",
        toolCallId: "call-1",
        toolName: "bash_exec",
      }],
    } satisfies MobileConversationMessage;

    expect(areMobileConversationMessagesEqual(message, changed)).toBe(false);
  });
});

function assistantMessage(): MobileConversationMessage {
  return {
    id: "session-1:1:assistant",
    parts: [{
      id: "text-1",
      kind: "text",
      text: "loaded",
    }],
    role: "assistant",
    sessionId: "session-1",
    text: "loaded",
    tools: [{
      id: "tool-1",
      input: "pwd",
      output: "/repo",
      status: "success",
      toolCallId: "call-1",
      toolName: "bash_exec",
    }],
  };
}
