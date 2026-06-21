// @vitest-environment jsdom
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AssistantReply } from "./Messages";
import type { AgentPayload, StatusMessage } from "../../mobileTypes";

const successStatus: StatusMessage = { tone: "success", text: "回复已返回" };

describe("AssistantReply", () => {
  it("keeps completed thinking collapsed until the user opens it", () => {
    render(
      <AssistantReply
        reply={agentReply({ message: "完成", thinking: "内部思考内容" })}
        status={successStatus}
      />,
    );

    const toggle = screen.getByRole("button", { name: "已思考" });
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByText("内部思考内容")).toBeNull();

    fireEvent.click(toggle);

    expect(toggle.getAttribute("aria-expanded")).toBe("true");
    expect(screen.getByText("内部思考内容")).toBeTruthy();
  });

  it("keeps tool card details collapsed until the user opens them", () => {
    render(
      <AssistantReply
        reply={agentReply({
          tools: [
            {
              id: "tool-1",
              input: JSON.stringify({ cmd: "pwd" }),
              output: "/repo",
              status: "success",
              toolCallId: "call-1",
              toolName: "bash_exec",
            },
          ],
        })}
        status={successStatus}
      />,
    );

    const toggle = screen.getByRole("button", { name: "已运行 pwd" });
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByText("/repo")).toBeNull();

    fireEvent.click(toggle);

    expect(toggle.getAttribute("aria-expanded")).toBe("true");
    expect(screen.getByText("/repo")).toBeTruthy();
  });
});

function agentReply(patch: Partial<AgentPayload>): AgentPayload {
  return {
    message: patch.message ?? "",
    mode: patch.mode,
    session_ended: patch.session_ended ?? false,
    session_id: patch.session_id ?? "session-1",
    thinking: patch.thinking,
    tools: patch.tools,
  };
}
