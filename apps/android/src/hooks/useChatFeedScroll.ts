import { useCallback, useLayoutEffect, useRef, useState } from "react";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "../mobileTypes";

const BOTTOM_THRESHOLD_PX = 48;
const LOAD_OLDER_THRESHOLD_PX = 32;
const SCROLL_ANCHOR_TOLERANCE_PX = 2;
const USER_SCROLL_LOCK_RELEASE_MS = 1500;
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

interface PostSendAnchor {
  messageId: string;
  targetScrollTop: number;
}

export function useChatFeedScroll(options: UseChatFeedScrollOptions) {
  const historySentinelRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLElement>(null);
  const userMessageRowsRef = useRef(new Map<string, HTMLDivElement>());
  const handledPostSendTokenRef = useRef<number | null>(null);
  const pendingPostSendRequestRef = useRef<PostSendFocusRequest | null>(null);
  const postSendAnchorRef = useRef<PostSendAnchor | null>(null);
  const postSendAnchorJustFocusedRef = useRef(false);
  const historyAnchorRef = useRef<HistoryAnchorSnapshot | null>(null);
  const olderLoadPendingRef = useRef(false);
  const previousSessionIdRef = useRef<string | undefined>(undefined);
  const previousScrollTopRef = useRef(0);
  const autoScrollRef = useRef(false);
  const isUserScrollingRef = useRef(false);
  const userScrollReleaseTimeoutRef = useRef<number | null>(null);
  const [showScrollDown, setShowScrollDown] = useState(false);
  const [trailingSpacerPx, setTrailingSpacerPxState] = useState(0);
  const [registeredUserRowVersion, setRegisteredUserRowVersion] = useState(0);
  const hasReply = Boolean(options.reply);

  const setTrailingSpacerPx = useCallback((value: number) => {
    setTrailingSpacerPxState(Math.max(0, Math.ceil(value)));
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
    if (previousSessionIdRef.current === currentSessionId) {
      return;
    }

    const hadPreviousSession = previousSessionIdRef.current !== undefined;
    previousSessionIdRef.current = currentSessionId;
    pendingPostSendRequestRef.current = null;
    postSendAnchorRef.current = null;
    postSendAnchorJustFocusedRef.current = false;
    historyAnchorRef.current = null;
    olderLoadPendingRef.current = false;
    autoScrollRef.current = false;
    releaseUserScrollLock();
    setTrailingSpacerPx(0);
    setShowScrollDown(false);
    if (hadPreviousSession && scrollRef.current) {
      scrollRef.current.scrollTop = 0;
      previousScrollTopRef.current = 0;
    }
  }, [options.sessionId, setTrailingSpacerPx]);

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
  }, [options.messages, options.postSendFocusRequest, registeredUserRowVersion, setTrailingSpacerPx]);

  useLayoutEffect(() => {
    compensateHistoryAnchor();
    syncPostSendAnchor();

    if (!options.loadingOlderHistory && autoScrollRef.current && (options.messages.length > 0 || hasReply)) {
      scrollToBottom("auto");
      return;
    }

    if (postSendAnchorRef.current) {
      setShowScrollDown(false);
      return;
    }

    syncBottomAffordance();
  }, [hasReply, options.loadingOlderHistory, options.messages, options.reply, options.statusTone]);

  useLayoutEffect(() => {
    if (!options.loadingOlderHistory) {
      olderLoadPendingRef.current = false;
    }
  }, [options.loadingOlderHistory]);

  useLayoutEffect(() => {
    return () => {
      clearUserScrollReleaseTimeout();
    };
  }, []);

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element || typeof ResizeObserver === "undefined") {
      return;
    }

    const resizeObserver = new ResizeObserver(() => {
      syncPostSendAnchor();
      if (autoScrollRef.current && !isUserScrollingRef.current) {
        scrollToBottom("auto");
        return;
      }
      syncBottomAffordance();
    });

    resizeObserver.observe(element);
    const contentElement = element.querySelector<HTMLElement>(CHAT_FEED_CONTENT_SELECTOR);
    if (contentElement) {
      resizeObserver.observe(contentElement);
    }
    return () => {
      resizeObserver.disconnect();
    };
  }, []);

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
    }, {
      root,
      threshold: 0.1,
    });

    observer.observe(target);
    return () => {
      observer.disconnect();
    };
  }, [options.hasOlderHistory, options.loadingOlderHistory, options.onLoadOlderHistory]);

  function handleScroll(): void {
    const element = scrollRef.current;
    if (!element) {
      return;
    }

    const previousScrollTop = previousScrollTopRef.current;
    previousScrollTopRef.current = element.scrollTop;
    if (autoScrollRef.current && element.scrollTop < previousScrollTop - SCROLL_ANCHOR_TOLERANCE_PX) {
      autoScrollRef.current = false;
    }

    maybeLoadOlderHistory(element);
    if (isAtBottom(element)) {
      autoScrollRef.current = true;
      setShowScrollDown(false);
      return;
    }

    setShowScrollDown(distanceFromBottom(element) > BOTTOM_THRESHOLD_PX);
  }

  function handleUserScrollIntent(): void {
    isUserScrollingRef.current = true;
    clearUserScrollReleaseTimeout();
    userScrollReleaseTimeoutRef.current = window.setTimeout(() => {
      isUserScrollingRef.current = false;
      userScrollReleaseTimeoutRef.current = null;
      if (autoScrollRef.current) {
        scrollToBottom("auto");
        return;
      }
      syncBottomAffordance();
    }, USER_SCROLL_LOCK_RELEASE_MS);
  }

  function resetScrollDown(): void {
    autoScrollRef.current = false;
    postSendAnchorRef.current = null;
    historyAnchorRef.current = null;
    setTrailingSpacerPx(0);
    setShowScrollDown(false);
  }

  function scrollToBottom(behavior: ScrollBehavior = "smooth"): void {
    const element = scrollRef.current;
    if (!element) {
      return;
    }

    releaseUserScrollLock();
    autoScrollRef.current = true;
    postSendAnchorRef.current = null;
    historyAnchorRef.current = null;
    setTrailingSpacerPx(0);
    setShowScrollDown(false);
    const nextScrollTop = Math.max(0, element.scrollHeight - element.clientHeight);
    element.scrollTo({ top: nextScrollTop, behavior });
    if (behavior !== "smooth") {
      previousScrollTopRef.current = nextScrollTop;
    }
  }

  function focusUserMessage(messageId: string): boolean {
    const element = scrollRef.current;
    const targetScrollTop = measureUserMessageTargetScrollTop(messageId);
    if (!element || targetScrollTop === null) {
      return false;
    }

    autoScrollRef.current = false;
    postSendAnchorRef.current = { messageId, targetScrollTop };
    postSendAnchorJustFocusedRef.current = true;
    setTrailingSpacerPx(requiredTrailingSpacerPx(element, targetScrollTop));
    element.scrollTo({ top: targetScrollTop, behavior: "auto" });
    previousScrollTopRef.current = targetScrollTop;
    setShowScrollDown(false);
    return true;
  }

  function syncPostSendAnchor(): void {
    const anchor = postSendAnchorRef.current;
    const element = scrollRef.current;
    if (!anchor || !element || autoScrollRef.current || isUserScrollingRef.current) {
      return;
    }
    if (postSendAnchorJustFocusedRef.current) {
      postSendAnchorJustFocusedRef.current = false;
      return;
    }

    const targetScrollTop = measureUserMessageTargetScrollTop(anchor.messageId);
    if (targetScrollTop === null) {
      postSendAnchorRef.current = null;
      return;
    }

    anchor.targetScrollTop = targetScrollTop;
    setTrailingSpacerPx(requiredTrailingSpacerPx(element, targetScrollTop));
    if (Math.abs(element.scrollTop - targetScrollTop) > SCROLL_ANCHOR_TOLERANCE_PX) {
      element.scrollTo({ top: targetScrollTop, behavior: "auto" });
      previousScrollTopRef.current = targetScrollTop;
    }
  }

  function measureUserMessageTargetScrollTop(messageId: string): number | null {
    const element = scrollRef.current;
    const row = userMessageRowsRef.current.get(messageId);
    if (!element || !row) {
      return null;
    }
    return element.scrollTop + (row.getBoundingClientRect().top - element.getBoundingClientRect().top);
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
    previousScrollTopRef.current = nextScrollTop;
    historyAnchorRef.current = null;
  }

  function syncBottomAffordance(): void {
    const element = scrollRef.current;
    if (!element) {
      return;
    }
    setShowScrollDown(distanceFromBottom(element) > BOTTOM_THRESHOLD_PX);
  }

  return {
    handleScroll,
    handleUserScrollIntent,
    historySentinelRef,
    registerUserMessageRow,
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
    trailingSpacerPx,
  };

  function clearUserScrollReleaseTimeout(): void {
    if (userScrollReleaseTimeoutRef.current === null) {
      return;
    }
    window.clearTimeout(userScrollReleaseTimeoutRef.current);
    userScrollReleaseTimeoutRef.current = null;
  }

  function releaseUserScrollLock(): void {
    clearUserScrollReleaseTimeout();
    isUserScrollingRef.current = false;
  }
}

function distanceFromBottom(element: HTMLElement): number {
  return Math.max(0, element.scrollHeight - element.scrollTop - element.clientHeight);
}

function isAtBottom(element: HTMLElement): boolean {
  return distanceFromBottom(element) <= BOTTOM_THRESHOLD_PX;
}

function hasUserMessage(messages: MobileConversationMessage[], messageId: string): boolean {
  return messages.some((message) => message.id === messageId && message.role === "user");
}

function requiredTrailingSpacerPx(element: HTMLElement, targetScrollTop: number): number {
  return Math.max(0, targetScrollTop + element.clientHeight - element.scrollHeight);
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
