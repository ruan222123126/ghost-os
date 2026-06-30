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
}

interface PostSendScrollRequest {
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

  it("scrolls to the bottom when a post-send request arrives", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 200 });
    const userMessage = message("pending:user:1710000000000", "user");
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);
    vi.mocked(feedElement().scrollTo).mockClear();

    rerender(
      <ScrollHarness
        messages={[userMessage]}
        metrics={metrics}
        postSendScrollRequest={postSendRequest()}
        statusTone="loading"
      />,
    );

    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 1000, behavior: "auto" });
    expect(latestSnapshot().showScrollDown).toBe(false);
  });

  it("keeps streaming replies at the bottom after sending", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 200 });
    const userMessage = message("pending:user:1710000000000", "user");
    const { rerender } = render(<ScrollHarness messages={[]} metrics={metrics} />);

    rerender(
      <ScrollHarness
        messages={[userMessage]}
        metrics={metrics}
        postSendScrollRequest={postSendRequest()}
        statusTone="loading"
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();

    metrics.scrollHeight = 1240;
    rerender(
      <ScrollHarness
        messages={[userMessage]}
        metrics={metrics}
        reply={reply("streaming reply")}
        statusTone="loading"
      />,
    );

    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 1240, behavior: "auto" });
  });

  it("does not pull the viewport down while the user is reading older messages", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 500 });
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

  it("resumes bottom follow when sending from a scrolled-up conversation", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 500 });
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
        postSendScrollRequest={postSendRequest()}
        statusTone="loading"
      />,
    );

    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 1180, behavior: "auto" });
    expect(latestSnapshot().showScrollDown).toBe(false);
  });

  it("loads older history at the top and preserves the current viewport", async () => {
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
    metrics.scrollTop = 0;
    fireEvent.scroll(feedElement());

    metrics.scrollHeight = 1200;
    rerender(
      <ScrollHarness
        messages={[message("session-2:1:user", "user")]}
        metrics={metrics}
        sessionId="session-2"
      />,
    );

    expect(metrics.scrollTop).toBe(1200);
  });

  it("keeps following the bottom when the feed resizes during streaming", () => {
    const metrics = feedMetrics({ clientHeight: 500, scrollHeight: 1000, scrollTop: 1000 });
    render(
      <ScrollHarness
        messages={[message("session-1:0:user", "user")]}
        metrics={metrics}
        reply={reply("streaming reply")}
        statusTone="loading"
      />,
    );
    vi.mocked(feedElement().scrollTo).mockClear();

    metrics.scrollHeight = 1120;
    notifyResize(feedElement());

    expect(feedElement().scrollTo).toHaveBeenLastCalledWith({ top: 1120, behavior: "auto" });
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
  hasOlderHistory?: boolean;
  loadingOlderHistory?: boolean;
  messages: MobileConversationMessage[];
  metrics: FeedMetrics;
  onLoadOlderHistory?: () => Promise<void>;
  postSendScrollRequest?: PostSendScrollRequest | null;
  reply?: AgentPayload;
  sessionId?: string;
  statusTone?: StatusMessage["tone"];
}) {
  const scroll = useChatFeedScroll({
    hasOlderHistory: props.hasOlderHistory,
    loadingOlderHistory: props.loadingOlderHistory,
    messages: props.messages,
    onLoadOlderHistory: props.onLoadOlderHistory,
    postSendScrollRequest: props.postSendScrollRequest,
    reply: props.reply,
    sessionId: props.sessionId,
    statusTone: props.statusTone ?? "idle",
  });
  hookSnapshots.push({
    showScrollDown: scroll.showScrollDown,
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
    />
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
  if (!vi.isMockFunction(element.scrollTo)) {
    element.scrollTo = vi.fn((options?: ScrollToOptions | number, y?: number) => {
      metrics.scrollTop = typeof options === "number" ? y ?? options : options?.top ?? metrics.scrollTop;
    });
  }
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

function postSendRequest(token = 1): PostSendScrollRequest {
  return { token };
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
