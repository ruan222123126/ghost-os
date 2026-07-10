import { useCallback, useLayoutEffect, useRef, useState } from "react";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "../mobileTypes";

const BOTTOM_THRESHOLD_PX = 48;
const HARD_BOTTOM_TOLERANCE_PX = 2;
const LOAD_OLDER_THRESHOLD_PX = 32;
const SCROLL_DIRECTION_TOLERANCE_PX = 2;
const USER_SCROLL_INTENT_RELEASE_MS = 240;
const USER_SCROLL_END_RELEASE_MS = 320;
const CHAT_FEED_CONTENT_SELECTOR = "[data-chat-feed-content]";

interface UseChatFeedScrollOptions {
  hasOlderHistory?: boolean;
  loadingOlderHistory?: boolean;
  messages: MobileConversationMessage[];
  onLoadOlderHistory?: () => Promise<void>;
  postSendFocusRequest?: PostSendFocusRequest | null;
  reply: AgentPayload | undefined;
  sessionId?: string;
  statusTone: StatusMessage["tone"];
}

interface PostSendFocusRequest {
  messageId: string;
  token: number;
}

interface HistoryAnchorSnapshot {
  messageCount: number;
  scrollHeight: number;
  scrollTop: number;
}

interface DynamicSpacerAnchor {
  viewportBottom: number;
}

export function useChatFeedScroll(options: UseChatFeedScrollOptions) {
  const historySentinelRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLElement>(null);
  const trailingSpacerRef = useRef<HTMLDivElement>(null);
  const userMessageRowsRef = useRef(new Map<string, HTMLDivElement>());
  const handledPostSendTokenRef = useRef<number | null>(null);
  const pendingPostSendRequestRef = useRef<PostSendFocusRequest | null>(null);
  const dynamicSpacerAnchorRef = useRef<DynamicSpacerAnchor | null>(null);
  const historyAnchorRef = useRef<HistoryAnchorSnapshot | null>(null);
  const olderLoadPendingRef = useRef(false);
  const previousSessionIdRef = useRef<string | undefined>(undefined);
  const previousScrollTopRef = useRef(0);
  const autoScrollRef = useRef(false);
  const smoothScrollInProgressRef = useRef(false);
  const userScrollIntentRef = useRef(false);
  const userScrollIntentTimeoutRef = useRef<number | null>(null);
  const trailingSpacerPxRef = useRef(0);
  const [showScrollDown, setShowScrollDown] = useState(false);
  const [trailingSpacerPx, setTrailingSpacerPxState] = useState(0);
  const [registeredUserRowVersion, setRegisteredUserRowVersion] = useState(0);
  const hasReply = Boolean(options.reply);
  const hasFeedContent = options.messages.length > 0 || hasReply || options.statusTone === "error";
  const isStreaming = options.statusTone === "loading";

  const setTrailingSpacerPx = useCallback((value: number) => {
    const nextValue = Math.max(0, Math.ceil(value));
    trailingSpacerPxRef.current = nextValue;
    if (trailingSpacerRef.current) {
      trailingSpacerRef.current.style.minHeight = `${nextValue}px`;
    }
    setTrailingSpacerPxState((current) => current === nextValue ? current : nextValue);
  }, []);

  const registerUserMessageRow = useCallback((messageId: string) => {
    return (node: HTMLDivElement | null) => {
      if (node) {
        userMessageRowsRef.current.set(messageId, node);
        if (pendingPostSendRequestRef.current?.messageId === messageId) {
          setRegisteredUserRowVersion((version) => version + 1);
        }
        return;
      }
      userMessageRowsRef.current.delete(messageId);
    };
  }, []);

  useLayoutEffect(() => {
    const currentSessionId = options.sessionId?.trim() || "";
    const previousSessionId = previousSessionIdRef.current;
    if (previousSessionId === currentSessionId) {
      return;
    }
    previousSessionIdRef.current = currentSessionId;
    if (previousSessionId === undefined || isPendingSessionPromotion(previousSessionId, currentSessionId)) {
      return;
    }

    pendingPostSendRequestRef.current = null;
    dynamicSpacerAnchorRef.current = null;
    historyAnchorRef.current = null;
    olderLoadPendingRef.current = false;
    autoScrollRef.current = false;
    smoothScrollInProgressRef.current = false;
    clearUserScrollIntent();
    setTrailingSpacerPx(0);
    setShowScrollDown(false);
    if (scrollRef.current) {
      scrollRef.current.scrollTop = 0;
      previousScrollTopRef.current = 0;
    }
  }, [options.messages, options.postSendFocusRequest, options.sessionId, setTrailingSpacerPx]);

  useLayoutEffect(() => {
    const request = options.postSendFocusRequest;
    if (!request || handledPostSendTokenRef.current === request.token) {
      pendingPostSendRequestRef.current = null;
      return;
    }
    if (!hasUserMessage(options.messages, request.messageId)) {
      pendingPostSendRequestRef.current = null;
      return;
    }
    if (focusUserMessage(request.messageId)) {
      handledPostSendTokenRef.current = request.token;
      pendingPostSendRequestRef.current = null;
      return;
    }
    pendingPostSendRequestRef.current = request;
  }, [options.messages, options.postSendFocusRequest, registeredUserRowVersion]);

  useLayoutEffect(() => {
    compensateHistoryAnchor();
    if (smoothScrollInProgressRef.current) {
      setShowScrollDown(false);
      return;
    }
    if (!options.loadingOlderHistory && autoScrollRef.current && hasFeedContent) {
      forceScrollToBottom("auto");
      return;
    }
    if (isStreaming) {
      shrinkDynamicSpacer();
    } else {
      dynamicSpacerAnchorRef.current = null;
    }
    syncBottomAffordance();
  }, [hasFeedContent, isStreaming, options.loadingOlderHistory, options.messages, options.reply, options.statusTone]);

  useLayoutEffect(() => {
    if (!options.loadingOlderHistory) {
      olderLoadPendingRef.current = false;
    }
  }, [options.loadingOlderHistory]);

  useLayoutEffect(() => () => clearUserScrollIntent(), []);

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element || typeof ResizeObserver === "undefined") {
      return;
    }

    const resizeObserver = new ResizeObserver(() => {
      if (smoothScrollInProgressRef.current) {
        setShowScrollDown(false);
        return;
      }
      if (autoScrollRef.current) {
        forceScrollToBottom("auto");
      } else if (isStreaming) {
        shrinkDynamicSpacer();
      }
      syncBottomAffordance();
    });
    resizeObserver.observe(element);
    const contentElement = element.querySelector<HTMLElement>(CHAT_FEED_CONTENT_SELECTOR);
    if (contentElement) {
      resizeObserver.observe(contentElement);
    }
    return () => resizeObserver.disconnect();
  }, [hasFeedContent, isStreaming]);

  useLayoutEffect(() => {
    const root = scrollRef.current;
    const target = historySentinelRef.current;
    if (!root || !target || typeof IntersectionObserver === "undefined") {
      return;
    }

    const observer = new IntersectionObserver((entries) => {
      if (entries[0]?.isIntersecting) {
        maybeLoadOlderHistory(root);
      }
    }, { root, threshold: 0.1 });
    observer.observe(target);
    return () => observer.disconnect();
  }, [options.hasOlderHistory, options.loadingOlderHistory, options.onLoadOlderHistory]);

  function handleScroll(): void {
    const element = scrollRef.current;
    if (!element) {
      return;
    }

    const previousScrollTop = previousScrollTopRef.current;
    previousScrollTopRef.current = element.scrollTop;
    const userInitiated = userScrollIntentRef.current;
    if (userInitiated) {
      scheduleUserScrollIntentRelease(USER_SCROLL_INTENT_RELEASE_MS);
    }
    const interruptedAutoScroll = (
      autoScrollRef.current
      && userInitiated
      && element.scrollTop < previousScrollTop - SCROLL_DIRECTION_TOLERANCE_PX
    );
    if (interruptedAutoScroll) {
      autoScrollRef.current = false;
      smoothScrollInProgressRef.current = false;
    }

    maybeLoadOlderHistory(element);
    if (interruptedAutoScroll) {
      syncBottomAffordance();
      return;
    }
    if (smoothScrollInProgressRef.current) {
      if (isAtHardBottom(element)) {
        smoothScrollInProgressRef.current = false;
        autoScrollRef.current = true;
        previousScrollTopRef.current = element.scrollTop;
      }
      setShowScrollDown(false);
      return;
    }
    if (userInitiated && isAtBottom(element)) {
      autoScrollRef.current = true;
      dynamicSpacerAnchorRef.current = null;
      setTrailingSpacerPx(0);
      setShowScrollDown(false);
      forceScrollToBottom("auto");
      return;
    }
    syncBottomAffordance();
  }

  function handleUserScrollStart(): void {
    interruptSmoothScroll();
    userScrollIntentRef.current = true;
    clearUserScrollIntentTimeout();
  }

  function handleUserScrollIntent(): void {
    interruptSmoothScroll();
    userScrollIntentRef.current = true;
    scheduleUserScrollIntentRelease(USER_SCROLL_INTENT_RELEASE_MS);
  }

  function handleUserScrollEnd(): void {
    scheduleUserScrollIntentRelease(USER_SCROLL_END_RELEASE_MS);
  }

  function resetScrollDown(): void {
    autoScrollRef.current = false;
    smoothScrollInProgressRef.current = false;
    dynamicSpacerAnchorRef.current = null;
    historyAnchorRef.current = null;
    setTrailingSpacerPx(0);
    setShowScrollDown(false);
  }

  function scrollToBottom(
    behavior: ScrollBehavior = "smooth",
    performSmoothScroll?: () => void,
  ): void {
    clearUserScrollIntent();
    autoScrollRef.current = true;
    smoothScrollInProgressRef.current = behavior === "smooth";
    dynamicSpacerAnchorRef.current = null;
    historyAnchorRef.current = null;
    setTrailingSpacerPx(0);
    setShowScrollDown(false);
    if (behavior === "smooth" && performSmoothScroll) {
      performSmoothScroll();
      return;
    }
    forceScrollToBottom(behavior);
  }

  function focusUserMessage(messageId: string): boolean {
    const element = scrollRef.current;
    const targetScrollTop = measureUserMessageTargetScrollTop(messageId);
    if (!element || targetScrollTop === null) {
      return false;
    }

    autoScrollRef.current = false;
    smoothScrollInProgressRef.current = false;
    const viewportBottom = targetScrollTop + element.clientHeight;
    dynamicSpacerAnchorRef.current = { viewportBottom };
    setTrailingSpacerPx(requiredTrailingSpacerPx(element, viewportBottom, trailingSpacerPxRef.current));
    element.scrollTo({ top: targetScrollTop, behavior: "auto" });
    previousScrollTopRef.current = targetScrollTop;
    setShowScrollDown(false);
    return true;
  }

  function shrinkDynamicSpacer(): void {
    const element = scrollRef.current;
    const anchor = dynamicSpacerAnchorRef.current;
    if (!element || !anchor || trailingSpacerPxRef.current === 0) {
      return;
    }

    const required = requiredTrailingSpacerPx(element, anchor.viewportBottom, trailingSpacerPxRef.current);
    if (required < trailingSpacerPxRef.current) {
      setTrailingSpacerPx(required);
    }
  }

  function measureUserMessageTargetScrollTop(messageId: string): number | null {
    const element = scrollRef.current;
    const row = userMessageRowsRef.current.get(messageId);
    if (!element || !row) {
      return null;
    }
    return element.scrollTop + row.getBoundingClientRect().top - element.getBoundingClientRect().top;
  }

  function maybeLoadOlderHistory(element: HTMLElement): void {
    if (
      !options.hasOlderHistory
      || options.loadingOlderHistory
      || olderLoadPendingRef.current
      || element.scrollTop > LOAD_OLDER_THRESHOLD_PX
      || !options.onLoadOlderHistory
    ) {
      return;
    }

    olderLoadPendingRef.current = true;
    autoScrollRef.current = false;
    historyAnchorRef.current = {
      messageCount: options.messages.length,
      scrollHeight: element.scrollHeight,
      scrollTop: element.scrollTop,
    };
    setShowScrollDown(true);
    void options.onLoadOlderHistory().catch(() => {
      olderLoadPendingRef.current = false;
      historyAnchorRef.current = null;
    });
  }

  function compensateHistoryAnchor(): void {
    const element = scrollRef.current;
    const anchor = historyAnchorRef.current;
    if (!element || !anchor) {
      return;
    }
    if (options.messages.length <= anchor.messageCount) {
      if (!options.loadingOlderHistory) {
        historyAnchorRef.current = null;
      }
      return;
    }

    const delta = element.scrollHeight - anchor.scrollHeight;
    if (delta <= 0) {
      historyAnchorRef.current = null;
      return;
    }

    const nextScrollTop = anchor.scrollTop + delta;
    withAutoScrollBehavior(element, () => {
      element.scrollTop = nextScrollTop;
    });
    const spacerAnchor = dynamicSpacerAnchorRef.current;
    if (spacerAnchor) {
      spacerAnchor.viewportBottom += delta;
    }
    previousScrollTopRef.current = nextScrollTop;
    historyAnchorRef.current = null;
  }

  function forceScrollToBottom(behavior: ScrollBehavior): void {
    const element = scrollRef.current;
    if (!element) {
      return;
    }
    const nextScrollTop = Math.max(0, element.scrollHeight - element.clientHeight);
    element.scrollTo({ top: nextScrollTop, behavior });
    if (behavior !== "smooth") {
      previousScrollTopRef.current = nextScrollTop;
    }
  }

  function syncBottomAffordance(): void {
    const element = scrollRef.current;
    if (element) {
      setShowScrollDown(distanceFromBottom(element) > BOTTOM_THRESHOLD_PX);
    }
  }

  function scheduleUserScrollIntentRelease(delayMs: number): void {
    clearUserScrollIntentTimeout();
    userScrollIntentTimeoutRef.current = window.setTimeout(() => {
      userScrollIntentRef.current = false;
      userScrollIntentTimeoutRef.current = null;
    }, delayMs);
  }

  function clearUserScrollIntentTimeout(): void {
    if (userScrollIntentTimeoutRef.current !== null) {
      window.clearTimeout(userScrollIntentTimeoutRef.current);
      userScrollIntentTimeoutRef.current = null;
    }
  }

  function clearUserScrollIntent(): void {
    clearUserScrollIntentTimeout();
    userScrollIntentRef.current = false;
  }

  function interruptSmoothScroll(): void {
    if (!smoothScrollInProgressRef.current) {
      return;
    }
    smoothScrollInProgressRef.current = false;
    autoScrollRef.current = false;
  }

  function isPendingSessionPromotion(previousSessionId: string, currentSessionId: string): boolean {
    const request = options.postSendFocusRequest;
    return previousSessionId === ""
      && currentSessionId !== ""
      && Boolean(request && hasUserMessage(options.messages, request.messageId));
  }

  return {
    handleScroll,
    handleUserScrollEnd,
    handleUserScrollIntent,
    handleUserScrollStart,
    historySentinelRef,
    registerUserMessageRow,
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
    trailingSpacerPx,
    trailingSpacerRef,
  };
}

function distanceFromBottom(element: HTMLElement): number {
  return Math.max(0, element.scrollHeight - element.scrollTop - element.clientHeight);
}

function isAtBottom(element: HTMLElement): boolean {
  return distanceFromBottom(element) <= BOTTOM_THRESHOLD_PX;
}

function isAtHardBottom(element: HTMLElement): boolean {
  return distanceFromBottom(element) <= HARD_BOTTOM_TOLERANCE_PX;
}

function hasUserMessage(messages: MobileConversationMessage[], messageId: string): boolean {
  return messages.some((message) => message.id === messageId && message.role === "user");
}

function requiredTrailingSpacerPx(element: HTMLElement, viewportBottom: number, currentSpacerPx: number): number {
  const contentHeightWithoutSpacer = Math.max(0, element.scrollHeight - currentSpacerPx);
  return Math.max(0, viewportBottom - contentHeightWithoutSpacer);
}

function withAutoScrollBehavior(element: HTMLElement, action: () => void): void {
  const previousScrollBehavior = element.style.scrollBehavior;
  element.style.scrollBehavior = "auto";
  try {
    action();
  } finally {
    element.style.scrollBehavior = previousScrollBehavior;
  }
}
