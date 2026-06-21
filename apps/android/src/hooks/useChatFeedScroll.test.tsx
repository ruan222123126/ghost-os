// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useChatFeedScroll } from "./useChatFeedScroll";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "../mobileTypes";

interface FeedMetrics {
  clientHeight: number;
  scrollHeight: number;
  scrollTop: number;
}

interface HookSnapshot {
  showScrollDown: boolean;
  trailingSpacerPx: number;
}

const hookSnapshots: HookSnapshot[] = [];
let rafCallbacks: FrameRequestCallback[] = [];

describe("useChatFeedScroll", () => {
  beforeEach(() => {
    hookSnapshots.length = 0;
    rafCallbacks = [];
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      rafCallbacks.push(callback);
      return rafCallbacks.length;
    });
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("does not focus initial historical messages", () => {
    const metrics = feedMetrics();
    render(
      <ScrollHarness
        messages={[message("session-1:0:user", "user")]}
        metrics={metrics}
        rowTops={{ "session-1:0:user": 120 }}
      />,
    );

    expect(feedElement().scrollTo).not.toHaveBeenCalledWith({ top: 120, behavior: "smooth" });
    expect(latestSnapshot().trailingSpacerPx).toBe(0);
  });

  it("focuses an appended local user message at its row top", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300 });
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);

    rerender(
      <ScrollHarness
        messages={[message("pending:user:1710000000000", "user")]}
        metrics={metrics}
        rowTops={{ "pending:user:1710000000000": 120 }}
      />,
    );
    flushRaf();

    expect(latestSnapshot().trailingSpacerPx).toBe(320);
    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 120, behavior: "smooth" });
  });

  it("does not focus appended historical user ids", () => {
    const metrics = feedMetrics();
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);

    rerender(
      <ScrollHarness
        messages={[message("session-1:3:user", "user")]}
        metrics={metrics}
        rowTops={{ "session-1:3:user": 120 }}
      />,
    );
    flushRaf();

    expect(feedElement().scrollTo).not.toHaveBeenCalledWith({ top: 120, behavior: "smooth" });
    expect(latestSnapshot().trailingSpacerPx).toBe(0);
  });

  it("calculates enough spacer to keep the user message at the viewport top", () => {
    const metrics = feedMetrics({ clientHeight: 640, scrollHeight: 460 });
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);

    rerender(
      <ScrollHarness
        messages={[message("session-1:user:1710000000000", "user")]}
        metrics={metrics}
        rowTops={{ "session-1:user:1710000000000": 180 }}
      />,
    );
    flushRaf();

    expect(latestSnapshot().trailingSpacerPx).toBe(360);
    expect(screen.getByTestId("trailing-spacer").style.height).toBe("360px");
  });

  it("clears spacer and resumes bottom follow after streaming content fills the anchored viewport", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300 });
    const localMessage = message("pending:user:1710000000000", "user");
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);
    rerender(
      <ScrollHarness
        messages={[localMessage]}
        metrics={metrics}
        rowTops={{ [localMessage.id]: 120 }}
      />,
    );
    flushRaf();
    expect(latestSnapshot().trailingSpacerPx).toBe(320);

    metrics.scrollHeight = 1020;
    rerender(
      <ScrollHarness
        messages={[localMessage]}
        metrics={metrics}
        reply={reply("streaming reply")}
        rowTops={{ [localMessage.id]: 120 }}
        statusTone="loading"
      />,
    );
    metrics.scrollHeight = 700;
    flushRaf();

    expect(latestSnapshot().trailingSpacerPx).toBe(0);
    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 700, behavior: "smooth" });
  });

  it("cancels the post-send lock on manual upward scroll and clamps downward scroll into spacer", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300 });
    const localMessage = message("pending:user:1710000000000", "user");
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);
    rerender(
      <ScrollHarness
        messages={[localMessage]}
        metrics={metrics}
        rowTops={{ [localMessage.id]: 120 }}
      />,
    );
    flushRaf();

    const feed = feedElement();
    vi.mocked(feed.scrollTo).mockClear();
    metrics.scrollTop = 220;
    fireEvent.scroll(feed);
    expect(feed.scrollTo).toHaveBeenLastCalledWith({ top: 120, behavior: "auto" });
    expect(latestSnapshot().trailingSpacerPx).toBe(320);

    metrics.scrollTop = 120;
    fireEvent.scroll(feed);
    vi.mocked(feed.scrollTo).mockClear();
    metrics.scrollTop = 80;
    fireEvent.scroll(feed);
    metrics.scrollTop = 220;
    fireEvent.scroll(feed);

    expect(latestSnapshot().trailingSpacerPx).toBe(0);
    expect(feed.scrollTo).not.toHaveBeenCalled();
  });
});

function ScrollHarness(props: {
  messages: MobileConversationMessage[];
  metrics: FeedMetrics;
  reply?: AgentPayload;
  rowTops?: Record<string, number>;
  statusTone?: StatusMessage["tone"];
}) {
  const scroll = useChatFeedScroll({
    messages: props.messages,
    reply: props.reply,
    statusTone: props.statusTone ?? "idle",
  });
  hookSnapshots.push({
    showScrollDown: scroll.showScrollDown,
    trailingSpacerPx: scroll.trailingSpacerPx,
  });

  return (
    <main
      data-testid="feed"
      ref={(node) => {
        if (!node) {
          return;
        }
        applyFeedMetrics(node, props.metrics);
        scroll.scrollRef.current = node;
      }}
      onScroll={scroll.handleScroll}
    >
      {props.messages.map((item) =>
        item.role === "user" ? (
          <div
            key={item.id}
            ref={(node) => {
              if (!node) {
                return;
              }
              applyRowTop(node, props.rowTops?.[item.id] ?? 0, props.metrics);
              scroll.registerUserMessageRow(item.id)(node);
            }}
          >
            {item.text}
          </div>
        ) : null,
      )}
      <div data-testid="trailing-spacer" aria-hidden="true" style={{ height: scroll.trailingSpacerPx }} />
    </main>
  );
}

function feedMetrics(overrides: Partial<FeedMetrics> = {}): FeedMetrics {
  return {
    clientHeight: 500,
    scrollHeight: 1000,
    scrollTop: 0,
    ...overrides,
  };
}

function applyFeedMetrics(element: HTMLElement, metrics: FeedMetrics): void {
  Object.defineProperty(element, "clientHeight", {
    configurable: true,
    get: () => metrics.clientHeight,
  });
  Object.defineProperty(element, "scrollHeight", {
    configurable: true,
    get: () => metrics.scrollHeight,
  });
  Object.defineProperty(element, "scrollTop", {
    configurable: true,
    get: () => metrics.scrollTop,
    set: (value: number) => {
      metrics.scrollTop = value;
    },
  });
  element.getBoundingClientRect = () => domRect({ top: 0 });
  element.scrollTo = vi.fn((options?: ScrollToOptions | number, y?: number) => {
    metrics.scrollTop = typeof options === "number" ? y ?? options : options?.top ?? metrics.scrollTop;
  });
}

function applyRowTop(element: HTMLElement, top: number, metrics: FeedMetrics): void {
  element.getBoundingClientRect = () => domRect({ top: top - metrics.scrollTop });
}

function domRect(overrides: Partial<DOMRect>): DOMRect {
  return {
    bottom: 0,
    height: 0,
    left: 0,
    right: 0,
    toJSON: () => ({}),
    top: 0,
    width: 0,
    x: 0,
    y: 0,
    ...overrides,
  };
}

function message(id: string, role: MobileConversationMessage["role"]): MobileConversationMessage {
  return {
    id,
    role,
    text: "hello",
  };
}

function reply(text: string): AgentPayload {
  return {
    message: text,
    session_ended: false,
    session_id: "session-1",
  };
}

function feedElement(): HTMLElement & { scrollTo: ReturnType<typeof vi.fn> } {
  return screen.getByTestId("feed") as HTMLElement & { scrollTo: ReturnType<typeof vi.fn> };
}

function latestSnapshot(): HookSnapshot {
  const snapshot = hookSnapshots[hookSnapshots.length - 1];
  if (!snapshot) {
    throw new Error("No hook snapshot recorded");
  }
  return snapshot;
}

function flushRaf(): void {
  const callbacks = rafCallbacks;
  rafCallbacks = [];
  act(() => {
    for (const callback of callbacks) {
      callback(performance.now());
    }
  });
}
