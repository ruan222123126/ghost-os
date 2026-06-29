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
  });

  it("renders streaming content as plain text before final markdown parse", () => {
    render(<AssistantMarkdownContent content="```ts\nconsole.log(1)" final={false} showCopyButton={false} />);

    expect(screen.queryByTestId("markstream-renderer")).toBeNull();
    const pre = document.querySelector("pre");
    expect(pre?.textContent).toContain("```ts");
    expect(pre?.textContent).toContain("console.log(1)");
  });
});
