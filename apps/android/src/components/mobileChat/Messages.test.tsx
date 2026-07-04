// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AssistantReply, ChatBubble, ConversationMessageList } from "./Messages";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "../../mobileTypes";

const successStatus: StatusMessage = { tone: "success", text: "回复已返回" };

const markdownMock = vi.hoisted(() => (
  vi.fn((_: { content: string; final?: boolean; showCopyButton?: boolean }) => null)
));

vi.mock("./AssistantMarkdownContent", () => ({
  AssistantMarkdownContent: (props: { content: string; final?: boolean; showCopyButton?: boolean }) => (
    <>
      {markdownMock(props)}
      <div data-testid="assistant-markdown-content">{props.content}</div>
    </>
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

  it("renders assistant text and tool cards in reply part order", () => {
    const { container } = render(
      <AssistantReply
        reply={agentReply({
          message: "先开始绘图。图片已经生成。",
          parts: [
            {
              id: "text-1",
              kind: "text",
              text: "先开始绘图。",
            },
            {
              id: "tool-1",
              kind: "tool",
              tool: {
                id: "tool-1",
                input: JSON.stringify({ prompt: "cat" }),
                output: "Generated 1 image(s).",
                status: "success",
                toolCallId: "call-1",
                toolName: "screen_action",
              },
            },
            {
              id: "text-2",
              kind: "text",
              text: "图片已经生成。",
            },
          ],
        })}
        status={successStatus}
      />,
    );

    const partContainer = container.querySelector(".assistant-reply-parts");
    expect(partContainer?.children).toHaveLength(3);
    expect(partContainer?.children[0]?.textContent).toContain("先开始绘图。");
    expect(partContainer?.children[1]?.querySelector("button.tool-card-button")).toBeTruthy();
    expect(partContainer?.children[2]?.textContent).toContain("图片已经生成。");
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

describe("ChatBubble", () => {
  it("renders selected skill without replacing user text", () => {
    render(
      <ChatBubble selectedSkill={{ id: "skill_release", name: "release_flow" }}>
        发布版本
      </ChatBubble>,
    );

    expect(screen.getByText("release_flow")).toBeTruthy();
    expect(screen.getByText("发布版本")).toBeTruthy();
  });
});

describe("ConversationMessageList", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the active reply after the optimistic user message", () => {
    const { container } = render(
      <ConversationMessageList
        messages={[conversationMessage("pending:user:1", "user", "先执行")]}
        reply={agentReply({ message: "正在处理" })}
        registerUserMessageRow={() => () => undefined}
        status={{ tone: "loading", text: "正在回复" }}
      />,
    );

    const rows = [...container.querySelectorAll(".conversation-list > .message-row")];
    expect(rows).toHaveLength(2);
    expect(rows[0]?.textContent).toContain("正在处理");
    expect(rows[1]?.textContent).toBe("先执行");
  });
});

function agentReply(patch: Partial<AgentPayload>): AgentPayload {
  return {
    message: patch.message ?? "",
    mode: patch.mode,
    parts: patch.parts,
    session_ended: patch.session_ended ?? false,
    session_id: patch.session_id ?? "session-1",
    thinking: patch.thinking,
    tools: patch.tools,
  };
}

function conversationMessage(
  id: string,
  role: MobileConversationMessage["role"],
  text: string,
): MobileConversationMessage {
  return {
    id,
    role,
    text,
  };
}
