import { useLayoutEffect, useRef, useState } from "react";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "../mobileTypes";

const SCROLL_DOWN_THRESHOLD_PX = 50;
const LOAD_OLDER_THRESHOLD_PX = 32;
const OLDER_LOAD_STABILIZATION_FRAMES = 2;
const CHAT_FEED_CONTENT_SELECTOR = "[data-chat-feed-content]";
const CHAT_FEED_ITEM_SELECTOR = "[data-chat-feed-item]";

interface UseChatFeedScrollOptions {
  hasOlderHistory?: boolean;
  loadingOlderHistory?: boolean;
  messages: MobileConversationMessage[];
  onLoadOlderHistory?: () => Promise<void>;
  postSendScrollRequest?: PostSendScrollRequest | null;
  reply: AgentPayload | undefined;
  sessionId?: string;
  statusTone: StatusMessage["tone"];
}

interface PostSendScrollRequest {
  token: number;
}

interface OlderLoadAnchor {
  element: HTMLElement | null;
  elementTop: number | null;
  stabilizationFramesRemaining: number;
  scrollHeight: number;
  scrollTop: number;
}

export function useChatFeedScroll(options: UseChatFeedScrollOptions) {
  const scrollRef = useRef<HTMLElement>(null);
  const handledPostSendTokenRef = useRef<number | null>(null);
  const autoFollowRef = useRef(true);
  const olderLoadPendingRef = useRef(false);
  const olderLoadAnchorRef = useRef<OlderLoadAnchor | null>(null);
  const olderLoadStabilizationFrameRef = useRef<number | null>(null);
  const previousSessionIdRef = useRef<string | undefined>(undefined);
  const [showScrollDown, setShowScrollDown] = useState(false);
  const reply = options.reply;
  const hasReply = Boolean(reply);

  useLayoutEffect(() => {
    const currentSessionId = options.sessionId?.trim() || "";
    if (previousSessionIdRef.current === currentSessionId) {
      return;
    }

    previousSessionIdRef.current = currentSessionId;
    olderLoadPendingRef.current = false;
    clearOlderLoadAnchor();
    autoFollowRef.current = true;
    setShowScrollDown(false);
    if (scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [options.sessionId]);

  useLayoutEffect(() => {
    if (restoreOlderLoadAnchor()) {
      return;
    }
    const request = options.postSendScrollRequest;
    if (!request || handledPostSendTokenRef.current === request.token) {
      return;
    }

    handledPostSendTokenRef.current = request.token;
    scrollToBottom("auto");
  }, [options.messages, options.postSendScrollRequest]);

  useLayoutEffect(() => {
    if (restoreOlderLoadAnchor()) {
      return;
    }
    if (autoFollowRef.current && (options.messages.length > 0 || hasReply)) {
      scrollToBottom(options.statusTone === "loading" ? "auto" : "smooth");
    }
  }, [hasReply, options.messages, options.statusTone, reply]);

  useLayoutEffect(() => {
    if (!options.loadingOlderHistory && olderLoadPendingRef.current) {
      olderLoadPendingRef.current = false;
      if (!restoreOlderLoadAnchor()) {
        clearOlderLoadAnchor();
      }
    }
  }, [options.loadingOlderHistory]);

  useLayoutEffect(() => {
    return () => {
      clearOlderLoadAnchor();
    };
  }, []);

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element || typeof ResizeObserver === "undefined") {
      return;
    }

    const resizeObserver = new ResizeObserver(() => {
      if (olderLoadAnchorRef.current && options.loadingOlderHistory) {
        return;
      }
      if (restoreOlderLoadAnchor()) {
        return;
      }
      if (autoFollowRef.current && (options.messages.length > 0 || hasReply)) {
        scrollToBottom("auto");
        return;
      }

      const shouldShow = shouldShowScrollDown(element);
      autoFollowRef.current = !shouldShow;
      setShowScrollDown(shouldShow);
    });

    resizeObserver.observe(element);
    const contentElement = element.querySelector<HTMLElement>(CHAT_FEED_CONTENT_SELECTOR);
    if (contentElement) {
      resizeObserver.observe(contentElement);
    }
    return () => {
      resizeObserver.disconnect();
    };
  }, [hasReply, options.loadingOlderHistory, options.messages.length, options.statusTone, reply]);

  function handleScroll(): void {
    const element = scrollRef.current;
    if (!element) {
      return;
    }

    maybeLoadOlderHistory(element);
    if (olderLoadAnchorRef.current || options.loadingOlderHistory) {
      autoFollowRef.current = false;
      setShowScrollDown(false);
      return;
    }

    const shouldShow = shouldShowScrollDown(element);
    autoFollowRef.current = !shouldShow;
    setShowScrollDown(shouldShow);
  }

  function resetScrollDown(): void {
    autoFollowRef.current = true;
    setShowScrollDown(false);
  }

  function scrollToBottom(behavior: ScrollBehavior = "smooth"): void {
    autoFollowRef.current = true;
    setShowScrollDown(false);
    if (scrollRef.current) {
      scrollRef.current.scrollTo({
        top: scrollRef.current.scrollHeight,
        behavior,
      });
    }
  }

  function maybeLoadOlderHistory(element: HTMLElement): void {
    if (
      !options.hasOlderHistory
      || options.loadingOlderHistory
      || olderLoadPendingRef.current
      || olderLoadAnchorRef.current
      || element.scrollTop > LOAD_OLDER_THRESHOLD_PX
      || !options.onLoadOlderHistory
    ) {
      return;
    }

    olderLoadPendingRef.current = true;
    olderLoadAnchorRef.current = {
      ...captureVisibleAnchor(element),
      stabilizationFramesRemaining: OLDER_LOAD_STABILIZATION_FRAMES,
      scrollHeight: element.scrollHeight,
      scrollTop: element.scrollTop,
    };
    autoFollowRef.current = false;
    void options.onLoadOlderHistory().catch(() => {
      olderLoadPendingRef.current = false;
      clearOlderLoadAnchor();
    });
  }

  function restoreOlderLoadAnchor(): boolean {
    const anchor = olderLoadAnchorRef.current;
    const element = scrollRef.current;
    if (!anchor || !element) {
      return false;
    }

    const previousScrollTop = element.scrollTop;
    const restored = anchor.scrollHeight === element.scrollHeight && anchor.scrollTop === element.scrollTop
      ? true
      : restoreAnchorPosition(element, anchor);
    if (!restored) {
      return false;
    }

    autoFollowRef.current = false;
    setShowScrollDown(shouldShowScrollDown(element));
    const stabilizationFramesRemaining = anchor.scrollHeight !== element.scrollHeight
      || Math.abs(element.scrollTop - previousScrollTop) > 0.5
      ? OLDER_LOAD_STABILIZATION_FRAMES
      : anchor.stabilizationFramesRemaining;

    if (options.loadingOlderHistory) {
      olderLoadAnchorRef.current = {
        ...anchor,
        stabilizationFramesRemaining,
        scrollHeight: element.scrollHeight,
        scrollTop: element.scrollTop,
      };
      return true;
    }

    if (stabilizationFramesRemaining > 0) {
      olderLoadAnchorRef.current = {
        ...anchor,
        stabilizationFramesRemaining,
        scrollHeight: element.scrollHeight,
        scrollTop: element.scrollTop,
      };
      scheduleOlderLoadAnchorStabilization();
      return true;
    }

    clearOlderLoadAnchor();
    return true;
  }

  function restoreAnchorPosition(element: HTMLElement, anchor: OlderLoadAnchor): boolean {
    if (anchor.element && anchor.element.isConnected && anchor.elementTop !== null) {
      const topDelta = anchor.element.getBoundingClientRect().top - anchor.elementTop;
      if (Math.abs(topDelta) > 0.5) {
        setScrollTopInstant(element, element.scrollTop + topDelta);
      }
      return true;
    }
    if (element.scrollHeight <= anchor.scrollHeight) {
      return false;
    }

    const delta = element.scrollHeight - anchor.scrollHeight;
    setScrollTopInstant(element, anchor.scrollTop + delta);
    return true;
  }

  function scheduleOlderLoadAnchorStabilization(): void {
    if (olderLoadStabilizationFrameRef.current !== null) {
      return;
    }

    olderLoadStabilizationFrameRef.current = window.requestAnimationFrame(() => {
      olderLoadStabilizationFrameRef.current = null;
      const anchor = olderLoadAnchorRef.current;
      if (!anchor) {
        return;
      }

      olderLoadAnchorRef.current = {
        ...anchor,
        stabilizationFramesRemaining: Math.max(anchor.stabilizationFramesRemaining - 1, 0),
      };
      if (!restoreOlderLoadAnchor()) {
        clearOlderLoadAnchor();
      }
    });
  }

  function clearOlderLoadAnchor(): void {
    olderLoadAnchorRef.current = null;
    if (olderLoadStabilizationFrameRef.current === null) {
      return;
    }

    window.cancelAnimationFrame(olderLoadStabilizationFrameRef.current);
    olderLoadStabilizationFrameRef.current = null;
  }

  return {
    handleScroll,
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
  };
}

function setScrollTopInstant(element: HTMLElement, scrollTop: number): void {
  element.scrollTop = scrollTop;
}

function captureVisibleAnchor(element: HTMLElement): Pick<OlderLoadAnchor, "element" | "elementTop"> {
  const feedTop = element.getBoundingClientRect().top;
  const items = element.querySelectorAll<HTMLElement>(CHAT_FEED_ITEM_SELECTOR);
  for (const item of items) {
    const rect = item.getBoundingClientRect();
    if (rect.bottom > feedTop) {
      return {
        element: item,
        elementTop: rect.top,
      };
    }
  }
  return {
    element: null,
    elementTop: null,
  };
}

function shouldShowScrollDown(element: HTMLElement): boolean {
  return element.scrollHeight - element.scrollTop - element.clientHeight > SCROLL_DOWN_THRESHOLD_PX;
}
