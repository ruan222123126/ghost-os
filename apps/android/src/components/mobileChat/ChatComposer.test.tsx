// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ChatComposer } from "./ChatComposer";

let measuredScrollHeight = 52;
const originalInnerHeightDescriptor = Object.getOwnPropertyDescriptor(window, "innerHeight");
const originalVisualViewportDescriptor = Object.getOwnPropertyDescriptor(window, "visualViewport");

interface MockVisualViewport extends EventTarget {
  height: number;
  offsetTop: number;
}

describe("ChatComposer", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    measuredScrollHeight = 52;
    restoreWindowViewportProperties();
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

  it("shows a stop button while loading without input text", () => {
    const onStop = vi.fn(async () => undefined);

    renderComposer("", { canStop: true, disabled: true, loading: true, onStop });

    fireEvent.click(screen.getByRole("button", { name: "停止生成" }));

    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("keeps the stop button visible but disabled while stopping", () => {
    renderComposer("", { canStop: false, disabled: true, loading: true, onStop: vi.fn(async () => undefined) });

    expect(screen.getByRole<HTMLButtonElement>("button", { name: "停止中" }).disabled).toBe(true);
  });

  it("raises the dock when the focused visual viewport is covered by the keyboard", async () => {
    const viewport = mockVisualViewport({ height: 500, offsetTop: 0 });
    mockAnimationFrame();
    Object.defineProperty(window, "innerHeight", { configurable: true, value: 800 });

    renderComposer("");

    fireEvent.focus(screen.getByPlaceholderText("问问 Ghost-OS"));

    await waitFor(() => {
      expect(composerDock().style.getPropertyValue("--composer-keyboard-inset")).toBe("300px");
    });

    viewport.height = 620;
    viewport.dispatchEvent(new Event("resize"));

    await waitFor(() => {
      expect(composerDock().style.getPropertyValue("--composer-keyboard-inset")).toBe("180px");
    });
  });
});

function mockTextareaScrollHeight(): void {
  vi.spyOn(HTMLTextAreaElement.prototype, "scrollHeight", "get").mockImplementation(function scrollHeight(
    this: HTMLTextAreaElement,
  ) {
    return this.classList.contains("composer-single-line-measure") ? measuredScrollHeight : 52;
  });
}

function renderComposer(value: string, overrides: Partial<Parameters<typeof ChatComposer>[0]> = {}) {
  return render(
    <ChatComposer
      disabled={false}
      loading={false}
      value={value}
      onChange={vi.fn()}
      onOpenSettings={vi.fn()}
      onSubmit={vi.fn(async () => undefined)}
      {...overrides}
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

function composerDock(): HTMLElement {
  const input = screen.getByPlaceholderText("问问 Ghost-OS");
  const dock = input.closest(".composer-dock");
  if (!(dock instanceof HTMLElement)) {
    throw new Error("composer dock not found");
  }
  return dock;
}

function mockVisualViewport(input: { height: number; offsetTop: number }) {
  const viewport = new EventTarget() as MockVisualViewport;
  viewport.height = input.height;
  viewport.offsetTop = input.offsetTop;
  Object.defineProperty(window, "visualViewport", { configurable: true, value: viewport });
  return viewport;
}

function mockAnimationFrame(): void {
  vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
    callback(0);
    return 1;
  });
  vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);
}

function restoreWindowViewportProperties(): void {
  if (originalInnerHeightDescriptor) {
    Object.defineProperty(window, "innerHeight", originalInnerHeightDescriptor);
  }
  if (originalVisualViewportDescriptor) {
    Object.defineProperty(window, "visualViewport", originalVisualViewportDescriptor);
  } else {
    Reflect.deleteProperty(window, "visualViewport");
  }
}
