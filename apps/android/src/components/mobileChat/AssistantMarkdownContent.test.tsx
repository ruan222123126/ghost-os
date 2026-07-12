// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantMarkdownContent } from "./AssistantMarkdownContent";

vi.mock("markstream-react", async () => {
  const React = await import("react");

  return {
    default: ({
      codeBlockProps,
      content,
      customId,
      final,
      htmlPolicy,
      typewriter,
    }: {
      codeBlockProps?: { showCopyButton?: boolean };
      content: string;
      customId?: string;
      final?: boolean;
      htmlPolicy?: string;
      typewriter?: boolean;
    }) => React.createElement(
      "div",
      {
        "data-custom-id": customId,
        "data-final": String(final),
        "data-html-policy": htmlPolicy,
        "data-show-copy": String(codeBlockProps?.showCopyButton),
        "data-testid": "markstream-renderer",
        "data-typewriter": String(typewriter),
      },
      content,
    ),
    setCustomComponents: vi.fn(),
  };
});

describe("AssistantMarkdownContent", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders content through Markstream", () => {
    render(<AssistantMarkdownContent content="# Heading" />);

    const renderer = screen.getByTestId("markstream-renderer");
    expect(renderer.textContent).toBe("# Heading");
    expect(renderer.getAttribute("data-custom-id")).toBe("ghost-os-mobile-assistant-markdown");
    expect(renderer.getAttribute("data-final")).toBe("true");
    expect(renderer.getAttribute("data-html-policy")).toBe("safe");
    expect(renderer.getAttribute("data-show-copy")).toBe("true");
  });

  it("adds display spacing between adjacent sentence outputs", () => {
    render(<AssistantMarkdownContent content="先检查项目。测试通过。" />);

    expect(screen.getByTestId("markstream-renderer").textContent).toBe("先检查项目。\n\n测试通过。");
  });

  it("renders streaming content through Markstream with a closed code fence", () => {
    render(<AssistantMarkdownContent content="```ts\nconsole.log(1)" final={false} showCopyButton={false} />);

    const renderer = screen.getByTestId("markstream-renderer");
    expect(renderer.textContent).toContain("```ts");
    expect(renderer.textContent).toContain("console.log(1)");
    expect(renderer.textContent?.endsWith("\n```")).toBe(true);
    expect(renderer.getAttribute("data-final")).toBe("false");
    expect(renderer.getAttribute("data-typewriter")).toBe("false");
  });

  it("uses the same display spacing for streaming markdown", () => {
    render(<AssistantMarkdownContent content="先检查项目。测试通过。" final={false} showCopyButton={false} />);

    expect(screen.getByTestId("markstream-renderer").textContent).toBe("先检查项目。\n\n测试通过。");
  });

  it("prebuilds a streaming table after the header row arrives", () => {
    render(<AssistantMarkdownContent content="| 平台 | 状态 |" final={false} showCopyButton={false} />);

    expect(screen.getByTestId("markstream-renderer").textContent).toBe(
      "| 平台 | 状态 |\n| --- | --- |",
    );
  });

  it("completes a partial table delimiter and data row", () => {
    const { rerender } = render(
      <AssistantMarkdownContent content={"| 平台 | 状态 |\n| --- |"} final={false} showCopyButton={false} />,
    );

    expect(screen.getByTestId("markstream-renderer").textContent).toBe(
      "| 平台 | 状态 |\n| --- | --- |",
    );

    rerender(
      <AssistantMarkdownContent
        content={"| 平台 | 状态 |\n| --- | --- |\n| Android"}
        final={false}
        showCopyButton={false}
      />,
    );
    expect(screen.getByTestId("markstream-renderer").textContent).toBe(
      "| 平台 | 状态 |\n| --- | --- |\n| Android |  |",
    );
  });
});
