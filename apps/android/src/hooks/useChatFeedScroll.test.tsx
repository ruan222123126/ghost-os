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

interface PostSendFocusRequest {
  messageId: string;
  token: number;
}

type ResizeObserverCallback = ConstructorParameters<typeof ResizeObserver>[0];

const hookSnapshots: HookSnapshot[] = [];
let rafCallbacks: FrameRequestCallback[] = [];
let resizeObservers: MockResizeObserver[] = [];

describe("useChatFeedScroll", () => {
  beforeEach(() => {
    hookSnapshots.length = 0;
    rafCallbacks = [];
    resizeObservers = [];
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      rafCallbacks.push(callback);
      return rafCallbacks.length;
    });
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
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

  it("focuses a requested post-send user message at its row top", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300 });
    const userMessage = message("pending:user:1710000000000", "user");
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);

    rerender(
      <ScrollHarness
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={postSendRequest(userMessage.id)}
        rowTops={{ [userMessage.id]: 120 }}
      />,
    );
    flushRaf();

    expect(latestSnapshot().trailingSpacerPx).toBe(320);
    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 120, behavior: "smooth" });
  });

  it("does not focus appended messages without a post-send request", () => {
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

  it("focuses requested user messages without depending on id shape", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300 });
    const userMessage = message("session-1:3:user", "user");
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);

    rerender(
      <ScrollHarness
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={postSendRequest(userMessage.id)}
        rowTops={{ [userMessage.id]: 120 }}
      />,
    );
    flushRaf();

    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 120, behavior: "smooth" });
  });

  it("does not focus the same post-send token more than once", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300 });
    const userMessage = message("pending:user:1710000000000", "user");
    const request = postSendRequest(userMessage.id);
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);

    rerender(
      <ScrollHarness
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={request}
        rowTops={{ [userMessage.id]: 120 }}
      />,
    );
    flushRaf();
    vi.mocked(feedElement().scrollTo).mockClear();

    rerender(
      <ScrollHarness
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={request}
        rowTops={{ [userMessage.id]: 120 }}
      />,
    );
    flushRaf();

    expect(feedElement().scrollTo).not.toHaveBeenCalled();
  });

  it("calculates enough spacer to keep the user message at the viewport top", () => {
    const metrics = feedMetrics({ clientHeight: 640, scrollHeight: 460 });
    const userMessage = message("session-1:user:1710000000000", "user");
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);

    rerender(
      <ScrollHarness
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={postSendRequest(userMessage.id)}
        rowTops={{ [userMessage.id]: 180 }}
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
        postSendFocusRequest={postSendRequest(localMessage.id)}
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
        postSendFocusRequest={postSendRequest(localMessage.id)}
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

  it("recomputes the locked spacer when the feed viewport height changes", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300 });
    const localMessage = message("pending:user:1710000000000", "user");
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);

    rerender(
      <ScrollHarness
        messages={[localMessage]}
        metrics={metrics}
        postSendFocusRequest={postSendRequest(localMessage.id)}
        rowTops={{ [localMessage.id]: 120 }}
      />,
    );
    flushRaf();

    expect(latestSnapshot().trailingSpacerPx).toBe(320);

    metrics.scrollHeight = 620;
    metrics.clientHeight = 640;
    notifyResize(feedElement());

    expect(latestSnapshot().trailingSpacerPx).toBe(460);
  });
});

class MockResizeObserver {
  readonly disconnect = vi.fn(() => {
    this.elements.clear();
  });
  readonly observe = vi.fn((element: Element) => {
    this.elements.add(element);
  });
  readonly unobserve = vi.fn((element: Element) => {
    this.elements.delete(element);
  });
  private readonly elements = new Set<Element>();

  constructor(private readonly callback: ResizeObserverCallback) {
    resizeObservers.push(this);
  }

  notify(element: Element): void {
    if (!this.elements.has(element)) {
      return;
    }

    this.callback([{ target: element } as ResizeObserverEntry], this as unknown as ResizeObserver);
  }
}

function ScrollHarness(props: {
  messages: MobileConversationMessage[];
  metrics: FeedMetrics;
  postSendFocusRequest?: PostSendFocusRequest | null;
  reply?: AgentPayload;
  rowTops?: Record<string, number>;
  statusTone?: StatusMessage["tone"];
}) {
  const scroll = useChatFeedScroll({
    messages: props.messages,
    postSendFocusRequest: props.postSendFocusRequest,
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

function postSendRequest(messageId: string, token = 1): PostSendFocusRequest {
  return { messageId, token };
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

function notifyResize(element: Element): void {
  act(() => {
    for (const observer of resizeObservers) {
      observer.notify(element);
    }
  });
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
