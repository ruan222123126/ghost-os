// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useChatFeedScroll } from "./useChatFeedScroll";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "../mobileTypes";

interface FeedMetrics {
  clientHeight: number;
  scrollHeight: number;
  scrollTop: number;
  top: number;
}

interface HookSnapshot {
  showScrollDown: boolean;
  trailingSpacerPx: number;
}

interface FeedItemMetrics {
  bottom: number;
  id: string;
  top: number;
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
    vi.useFakeTimers();
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      rafCallbacks.push(callback);
      return rafCallbacks.length;
    });
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("focuses a requested post-send user message at its row top", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300, scrollTop: 0 });
    const userMessage = message("pending:user:1710000000000", "user");
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);
    vi.mocked(feedElement().scrollTo).mockClear();

    rerender(
      <ScrollHarness
        feedItems={[feedItem(userMessage.id, 120, 200)]}
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={postSendRequest(userMessage.id)}
        statusTone="loading"
      />,
    );
    flushRaf();

    expect(latestSnapshot().trailingSpacerPx).toBe(320);
    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 120, behavior: "auto" });
    expect(latestSnapshot().showScrollDown).toBe(false);
  });

  it("shrinks streaming whitespace with content growth and retains the final remainder", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300, scrollTop: 0 });
    const userMessage = message("pending:user:1710000000000", "user");
    const { rerender } = render(
      <ScrollHarness
        feedItems={[feedItem(userMessage.id, 120, 200)]}
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={postSendRequest(userMessage.id)}
        reply={reply("partial")}
        statusTone="loading"
      />,
    );

    expect(latestSnapshot().trailingSpacerPx).toBe(320);

    metrics.scrollHeight = 720;
    rerender(
      <ScrollHarness
        feedItems={[feedItem(userMessage.id, 120, 200)]}
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={postSendRequest(userMessage.id)}
        reply={reply("partial response grew")}
        statusTone="loading"
      />,
    );
    expect(latestSnapshot().trailingSpacerPx).toBe(220);

    rerender(
      <ScrollHarness
        feedItems={[feedItem(userMessage.id, 120, 200)]}
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={postSendRequest(userMessage.id)}
        reply={reply("complete")}
        statusTone="success"
      />,
    );
    expect(latestSnapshot().trailingSpacerPx).toBe(220);
  });

  it("does not focus appended messages without a post-send request", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300, scrollTop: 0 });
    const userMessage = message("pending:user:1710000000000", "user");
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);

    rerender(
      <ScrollHarness
        feedItems={[feedItem(userMessage.id, 120, 200)]}
        messages={[userMessage]}
        metrics={metrics}
      />,
    );
    flushRaf();

    expect(feedElement().scrollTo).not.toHaveBeenCalledWith({ top: -120, behavior: "smooth" });
    expect(latestSnapshot().trailingSpacerPx).toBe(0);
  });

  it("does not pull the viewport down while the user is reading older messages", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 0 });
    const { rerender } = render(
      <ScrollHarness
        messages={[message("session-1:0:user", "user")]}
        metrics={metrics}
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();

    metrics.scrollTop = 120;
    fireEvent.scroll(feedElement());
    expect(latestSnapshot().showScrollDown).toBe(true);

    metrics.scrollHeight = 1200;
    rerender(
      <ScrollHarness
        messages={[message("session-1:0:user", "user")]}
        metrics={metrics}
        reply={reply("background stream")}
        statusTone="loading"
      />,
    );

    expect(feedElement().scrollTo).not.toHaveBeenCalled();
  });

  it("does not enable bottom follow from a programmatic scroll event", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 500 });
    const { rerender } = render(
      <ScrollHarness messages={[message("session-1:0:user", "user")]} metrics={metrics} />,
    );
    fireEvent.scroll(feedElement());
    vi.mocked(feedElement().scrollTo).mockClear();

    metrics.scrollHeight = 1120;
    rerender(
      <ScrollHarness
        messages={[message("session-1:0:user", "user")]}
        metrics={metrics}
        reply={reply("streaming reply")}
        statusTone="loading"
      />,
    );

    expect(feedElement().scrollTo).not.toHaveBeenCalled();
  });

  it("anchors a new user message without enabling automatic bottom follow", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 0 });
    const existingMessage = message("session-1:0:user", "user");
    const followUp = message("session-1:1:user", "user");
    const { rerender } = render(<ScrollHarness messages={[existingMessage]} metrics={metrics} />);
    vi.mocked(feedElement().scrollTo).mockClear();

    metrics.scrollTop = 120;
    fireEvent.scroll(feedElement());
    expect(latestSnapshot().showScrollDown).toBe(true);

    metrics.scrollHeight = 1180;
    rerender(
      <ScrollHarness
        messages={[existingMessage, followUp]}
        metrics={metrics}
        feedItems={[feedItem(followUp.id, 140, 220)]}
        postSendFocusRequest={postSendRequest(followUp.id)}
        statusTone="loading"
      />,
    );
    flushRaf();

    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 260, behavior: "auto" });
    expect(latestSnapshot().showScrollDown).toBe(true);
  });

  it("loads older history at the top and compensates scrollTop", async () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 0 });
    const loadOlderHistory = vi.fn(async () => undefined);
    const latest = message("session-1:1:user", "user");
    const older = message("session-1:0:user", "user");
    const { rerender } = render(
      <ScrollHarness
        hasOlderHistory
        messages={[latest]}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();
    metrics.scrollTop = 0;

    fireEvent.scroll(feedElement());

    expect(loadOlderHistory).toHaveBeenCalledTimes(1);

    metrics.scrollHeight = 1300;
    rerender(
      <ScrollHarness
        hasOlderHistory={false}
        messages={[older, latest]}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );
    flushRaf();

    expect(metrics.scrollTop).toBe(300);
    expect(feedElement().scrollTo).not.toHaveBeenCalled();
  });

  it("keeps the viewport stable when prepended history changes total height", async () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 0 });
    const loadOlderHistory = vi.fn(async () => undefined);
    const latest = message("session-1:1:user", "user");
    const older = message("session-1:0:user", "user");
    const loadedMessages = [older, latest];
    const { rerender } = render(
      <ScrollHarness
        feedItems={[feedItem("latest", 0, 80)]}
        hasOlderHistory
        messages={[latest]}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();
    metrics.scrollTop = 0;

    fireEvent.scroll(feedElement());

    expect(loadOlderHistory).toHaveBeenCalledTimes(1);

    metrics.scrollHeight = 1400;
    rerender(
      <ScrollHarness
        feedItems={[
          feedItem("older", 0, 220),
          feedItem("latest", 260, 340),
        ]}
        hasOlderHistory={false}
        messages={loadedMessages}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );

    expect(metrics.scrollTop).toBe(260);
    expect(feedElement().scrollTo).not.toHaveBeenCalled();
  });

  it("keeps the same visible row anchored when prepended history resizes after render", async () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 0 });
    const loadOlderHistory = vi.fn(async () => undefined);
    const latest = message("session-1:1:user", "user");
    const older = message("session-1:0:user", "user");
    const loadedMessages = [older, latest];
    const { rerender } = render(
      <ScrollHarness
        feedItems={[feedItem("latest", 0, 80)]}
        hasOlderHistory
        messages={[latest]}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();
    metrics.scrollTop = 0;

    fireEvent.scroll(feedElement());

    metrics.scrollHeight = 1300;
    rerender(
      <ScrollHarness
        feedItems={[
          feedItem("older", 0, 220),
          feedItem("latest", 300, 380),
        ]}
        hasOlderHistory={false}
        messages={loadedMessages}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );

    expect(metrics.scrollTop).toBe(300);
    expect(feedElement().style.scrollBehavior).toBe("");

    metrics.scrollHeight = 1420;
    rerender(
      <ScrollHarness
        feedItems={[
          feedItem("older", -300, 120),
          feedItem("latest", 120, 200),
        ]}
        hasOlderHistory={false}
        messages={loadedMessages}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );
    notifyResize(feedContentElement());

    expect(metrics.scrollTop).toBe(420);
  });

  it("temporarily disables smooth behavior while compensating prepended history", async () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 0 });
    const loadOlderHistory = vi.fn(async () => undefined);
    const latest = message("session-1:1:user", "user");
    const older = message("session-1:0:user", "user");
    const { rerender } = render(
      <ScrollHarness
        hasOlderHistory
        messages={[latest]}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );
    feedElement().style.scrollBehavior = "smooth";

    fireEvent.scroll(feedElement());

    metrics.scrollHeight = 1300;
    rerender(
      <ScrollHarness
        hasOlderHistory={false}
        messages={[older, latest]}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );

    expect(metrics.scrollTop).toBe(300);
    expect(feedElement().style.scrollBehavior).toBe("smooth");
  });

  it("keeps the older-history anchor while the load request is still pending", async () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 0 });
    const loadOlderHistory = vi.fn(() => new Promise<void>(() => undefined));
    const latest = message("session-1:1:user", "user");
    const older = message("session-1:0:user", "user");
    const loadedMessages = [older, latest];
    const { rerender } = render(
      <ScrollHarness
        feedItems={[feedItem("latest", 0, 80)]}
        hasOlderHistory
        messages={[latest]}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();
    metrics.scrollTop = 0;

    fireEvent.scroll(feedElement());

    expect(loadOlderHistory).toHaveBeenCalledTimes(1);

    metrics.scrollHeight = 1100;
    rerender(
      <ScrollHarness
        feedItems={[feedItem("latest", 0, 80)]}
        hasOlderHistory
        loadingOlderHistory
        messages={[latest]}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );
    notifyResize(feedContentElement());
    flushRaf();

    metrics.scrollHeight = 1360;
    rerender(
      <ScrollHarness
        feedItems={[
          feedItem("older", 0, 220),
          feedItem("latest", 260, 340),
        ]}
        hasOlderHistory={false}
        messages={loadedMessages}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );

    expect(metrics.scrollTop).toBe(260);
    expect(feedElement().scrollTo).not.toHaveBeenCalled();
  });

  it("does not load older history while away from the top", async () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 300 });
    const loadOlderHistory = vi.fn(async () => undefined);
    const latest = message("session-1:1:user", "user");
    render(
      <ScrollHarness
        feedItems={[feedItem("latest", 0, 80)]}
        hasOlderHistory
        messages={[latest]}
        metrics={metrics}
        onLoadOlderHistory={loadOlderHistory}
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();

    fireEvent.scroll(feedElement());

    expect(loadOlderHistory).not.toHaveBeenCalled();
    expect(feedElement().scrollTo).not.toHaveBeenCalled();
  });

  it("resets bottom follow when switching sessions", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 0 });
    const { rerender } = render(
      <ScrollHarness
        messages={[message("session-1:1:user", "user")]}
        metrics={metrics}
        sessionId="session-1"
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();
    metrics.scrollTop = 500;
    fireEvent.scroll(feedElement());

    metrics.scrollHeight = 1200;
    rerender(
      <ScrollHarness
        messages={[message("session-2:1:user", "user")]}
        metrics={metrics}
        sessionId="session-2"
      />,
    );
    flushRaf();

    expect(metrics.scrollTop).toBe(0);
  });

  it("preserves the post-send anchor when a new conversation receives its session id", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 300, scrollTop: 0 });
    const userMessage = message("pending:user:1710000000000", "user");
    const request = postSendRequest(userMessage.id);
    const { rerender } = render(
      <ScrollHarness
        feedItems={[feedItem(userMessage.id, 120, 200)]}
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={request}
        reply={reply("partial")}
        statusTone="loading"
      />,
    );

    expect(metrics.scrollTop).toBe(120);
    expect(latestSnapshot().trailingSpacerPx).toBe(320);

    rerender(
      <ScrollHarness
        feedItems={[feedItem(userMessage.id, 0, 80)]}
        messages={[userMessage]}
        metrics={metrics}
        postSendFocusRequest={request}
        reply={reply("partial response")}
        sessionId="session-1"
        statusTone="loading"
      />,
    );

    expect(metrics.scrollTop).toBe(120);
    expect(latestSnapshot().trailingSpacerPx).toBe(320);
  });

  it("keeps following the bottom when the feed resizes during streaming", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 500 });
    render(
      <ScrollHarness
        messages={[message("session-1:0:user", "user")]}
        metrics={metrics}
        reply={reply("streaming reply")}
        statusTone="loading"
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();
    fireEvent.wheel(feedElement());
    fireEvent.scroll(feedElement());

    metrics.scrollHeight = 1120;
    notifyResize(feedElement());
    flushRaf();

    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 620, behavior: "auto" });
  });

  it("cancels bottom follow as soon as the user scrolls upward", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 500 });
    render(
      <ScrollHarness
        messages={[message("session-1:0:user", "user")]}
        metrics={metrics}
        reply={reply("streaming reply")}
        statusTone="loading"
      />,
    );
    fireEvent.wheel(feedElement());
    fireEvent.scroll(feedElement());
    vi.mocked(feedElement().scrollTo).mockClear();

    fireEvent.wheel(feedElement());
    metrics.scrollTop = 470;
    fireEvent.scroll(feedElement());
    metrics.scrollHeight = 1120;
    notifyResize(feedElement());

    expect(feedElement().scrollTo).not.toHaveBeenCalled();
  });

  it("releases bottom follow on a one-pixel upward scroll", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 500 });
    render(
      <ScrollHarness
        messages={[message("session-1:0:user", "user")]}
        metrics={metrics}
        reply={reply("streaming reply")}
        statusTone="loading"
      />,
    );
    fireEvent.wheel(feedElement());
    fireEvent.scroll(feedElement());
    vi.mocked(feedElement().scrollTo).mockClear();

    fireEvent.wheel(feedElement());
    metrics.scrollTop = 499;
    fireEvent.scroll(feedElement());
    metrics.scrollHeight = 1120;
    notifyResize(feedElement());

    expect(metrics.scrollTop).toBe(499);
    expect(feedElement().scrollTo).not.toHaveBeenCalled();
  });

  it("does not enable bottom follow merely inside the scroll-down button threshold", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 440 });
    render(
      <ScrollHarness
        messages={[message("session-1:0:user", "user")]}
        metrics={metrics}
        reply={reply("streaming reply")}
        statusTone="loading"
      />,
    );
    fireEvent.wheel(feedElement());
    fireEvent.scroll(feedElement());
    vi.mocked(feedElement().scrollTo).mockClear();

    metrics.scrollTop = 455;
    fireEvent.scroll(feedElement());
    metrics.scrollHeight = 1120;
    notifyResize(feedElement());

    expect(metrics.scrollTop).toBe(455);
    expect(feedElement().scrollTo).not.toHaveBeenCalled();
  });

  it("does not interrupt an in-progress virtual smooth scroll when rows resize", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 120 });
    const smoothScrollToBottom = vi.fn();
    render(
      <ScrollHarness
        messages={[message("session-1:0:user", "user")]}
        metrics={metrics}
        showScrollDownControl
        smoothScrollToBottom={smoothScrollToBottom}
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();

    fireEvent.scroll(feedElement());
    fireEvent.click(screen.getByRole("button", { name: "scroll bottom" }));

    expect(smoothScrollToBottom).toHaveBeenCalledOnce();
    expect(feedElement().scrollTo).not.toHaveBeenCalled();

    metrics.scrollTop = 300;
    fireEvent.scroll(feedElement());

    metrics.scrollHeight = 1240;
    notifyResize(feedElement());

    expect(feedElement().scrollTo).not.toHaveBeenCalled();
    expect(latestSnapshot().showScrollDown).toBe(false);

    metrics.scrollTop = 740;
    fireEvent.scroll(feedElement());
    metrics.scrollHeight = 1360;
    notifyResize(feedElement());

    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 860, behavior: "auto" });
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
  feedItems?: FeedItemMetrics[];
  hasOlderHistory?: boolean;
  loadingOlderHistory?: boolean;
  messages: MobileConversationMessage[];
  metrics: FeedMetrics;
  onLoadOlderHistory?: () => Promise<void>;
  postSendFocusRequest?: PostSendFocusRequest | null;
  reply?: AgentPayload;
  sessionId?: string;
  showScrollDownControl?: boolean;
  smoothScrollToBottom?: () => void;
  statusTone?: StatusMessage["tone"];
}) {
  const scroll = useChatFeedScroll({
    hasOlderHistory: props.hasOlderHistory,
    loadingOlderHistory: props.loadingOlderHistory,
    messages: props.messages,
    onLoadOlderHistory: props.onLoadOlderHistory,
    postSendFocusRequest: props.postSendFocusRequest,
    reply: props.reply,
    sessionId: props.sessionId,
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
      onPointerCancel={scroll.handleUserScrollEnd}
      onPointerDown={scroll.handleUserScrollStart}
      onPointerUp={scroll.handleUserScrollEnd}
      onTouchCancel={scroll.handleUserScrollEnd}
      onTouchEnd={scroll.handleUserScrollEnd}
      onTouchMove={scroll.handleUserScrollIntent}
      onTouchStart={scroll.handleUserScrollStart}
      onWheel={scroll.handleUserScrollIntent}
    >
      <div data-testid="feed-content" data-chat-feed-content="">
        {props.feedItems?.map((item) => (
          <div
            key={item.id}
            ref={(node) => {
              scroll.registerUserMessageRow(item.id)(node);
              if (!node) {
                return;
              }
              applyFeedItemMetrics(node, item);
            }}
            data-chat-feed-item=""
            data-history-anchor-key={item.id}
          />
        ))}
      </div>
      <div data-testid="history-sentinel" ref={scroll.historySentinelRef} />
      <div data-testid="trailing-spacer" ref={scroll.trailingSpacerRef} style={{ minHeight: scroll.trailingSpacerPx }} />
      {props.showScrollDownControl ? (
        <button
          type="button"
          onClick={() => scroll.scrollToBottom("smooth", props.smoothScrollToBottom)}
          aria-label="scroll bottom"
        />
      ) : null}
    </main>
  );
}

function feedMetrics(overrides: Partial<FeedMetrics> = {}): FeedMetrics {
  return {
    clientHeight: 500,
    scrollHeight: 1000,
    scrollTop: 0,
    top: 0,
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
  element.getBoundingClientRect = () => domRect({ bottom: metrics.top + metrics.clientHeight, top: metrics.top });
  if (!vi.isMockFunction(element.scrollTo)) {
    element.scrollTo = vi.fn((options?: ScrollToOptions | number, y?: number) => {
      metrics.scrollTop = typeof options === "number" ? y ?? options : options?.top ?? metrics.scrollTop;
    });
  }
}

function applyFeedItemMetrics(element: HTMLElement, metrics: FeedItemMetrics): void {
  element.getBoundingClientRect = () => domRect({
    bottom: metrics.bottom,
    top: metrics.top,
  });
}

function domRect(input: { bottom: number; top: number }): DOMRect {
  return {
    bottom: input.bottom,
    height: input.bottom - input.top,
    left: 0,
    right: 0,
    toJSON: () => ({}),
    top: input.top,
    width: 0,
    x: 0,
    y: input.top,
  };
}

function message(id: string, role: MobileConversationMessage["role"]): MobileConversationMessage {
  return {
    id,
    role,
    text: "hello",
  };
}

function feedItem(id: string, top: number, bottom: number): FeedItemMetrics {
  return { bottom, id, top };
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

function feedContentElement(): HTMLElement {
  return screen.getByTestId("feed-content");
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
