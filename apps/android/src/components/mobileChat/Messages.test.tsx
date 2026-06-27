// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AssistantReply } from "./Messages";
import type { AgentPayload, StatusMessage } from "../../mobileTypes";

const successStatus: StatusMessage = { tone: "success", text: "回复已返回" };

const markdownMock = vi.hoisted(() => (
  vi.fn((_: { content: string; final?: boolean; showCopyButton?: boolean }) => null)
));

vi.mock("./AssistantMarkdownContent", () => ({
  AssistantMarkdownContent: (props: { content: string; final?: boolean; showCopyButton?: boolean }) => (
    markdownMock(props)
  ),
}));

describe("AssistantReply", () => {
  beforeEach(() => {
    markdownMock.mockClear();
  });

  afterEach(() => {
    cleanup();
  });

  it("opens streaming thinking when thinking text first appears", () => {
    render(
      <AssistantReply
        reply={agentReply({ thinking: "内部思考内容" })}
        status={{ tone: "loading", text: "思考中" }}
      />,
    );

    const toggle = screen.getByRole("button", { name: /正在思考/ });
    expect(toggle.getAttribute("aria-expanded")).toBe("true");
    expect(screen.getByText("内部思考内容")).toBeTruthy();
  });

  it("closes streaming thinking when answer text starts", () => {
    const { rerender } = render(
      <AssistantReply
        reply={agentReply({ thinking: "内部思考内容" })}
        status={{ tone: "loading", text: "思考中" }}
      />,
    );
    expect(screen.getByRole("button", { name: /正在思考/ }).getAttribute("aria-expanded")).toBe("true");

    rerender(
      <AssistantReply
        reply={agentReply({ message: "完成", thinking: "内部思考内容" })}
        status={{ tone: "loading", text: "生成中" }}
      />,
    );

    expect(screen.getByRole("button", { name: /正在思考/ }).getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByText("内部思考内容")).toBeNull();
  });

  it("closes streaming thinking when thinking completes without answer text", () => {
    const { rerender } = render(
      <AssistantReply
        reply={agentReply({ thinking: "内部思考内容" })}
        status={{ tone: "loading", text: "思考中" }}
      />,
    );
    expect(screen.getByRole("button", { name: /正在思考/ }).getAttribute("aria-expanded")).toBe("true");

    rerender(
      <AssistantReply
        reply={agentReply({ thinking: "内部思考内容" })}
        status={successStatus}
      />,
    );

    expect(screen.getByRole("button", { name: "已思考" }).getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByText("内部思考内容")).toBeNull();
  });

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

  it("passes streaming state to markdown renderer", () => {
    render(
      <AssistantReply
        reply={agentReply({ message: "partial" })}
        status={{ tone: "loading", text: "正在回复" }}
      />,
    );

    expect(markdownMock).toHaveBeenCalledWith(expect.objectContaining({
      content: "partial",
      final: false,
      showCopyButton: false,
    }));
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
