// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ChatComposer } from "./ChatComposer";

let measuredScrollHeight = 52;

describe("ChatComposer", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    measuredScrollHeight = 52;
  });

  it("switches to multiline when single-line layout overflows", () => {
    mockTextareaScrollHeight();
    measuredScrollHeight = 76;

    renderComposer("这是十一位中文输入");

    expect(composerShell().classList.contains("is-multiline")).toBe(true);
  });

  it("keeps long text single-line when measured layout still fits", () => {
    mockTextareaScrollHeight();
    measuredScrollHeight = 52;

    renderComposer("abcdefghijklmnopqrstuvwxyzabcdefghi");

    expect(composerShell().classList.contains("is-multiline")).toBe(false);
  });
});

function mockTextareaScrollHeight(): void {
  vi.spyOn(HTMLTextAreaElement.prototype, "scrollHeight", "get").mockImplementation(function scrollHeight(
    this: HTMLTextAreaElement,
  ) {
    return this.classList.contains("composer-single-line-measure") ? measuredScrollHeight : 52;
  });
}

function renderComposer(value: string) {
  return render(
    <ChatComposer
      disabled={false}
      loading={false}
      value={value}
      onChange={vi.fn()}
      onOpenSettings={vi.fn()}
      onSubmit={vi.fn(async () => undefined)}
    />,
  );
}

function composerShell(): HTMLElement {
  const input = screen.getByPlaceholderText("问问 Ghost-OS");
  const shell = input.closest(".composer-shell");
  if (!(shell instanceof HTMLElement)) {
    throw new Error("composer shell not found");
  }
  return shell;
}
