import { useCallback, useLayoutEffect, useRef, useState } from "react";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "../mobileTypes";

const SCROLL_DOWN_THRESHOLD_PX = 50;
const SCROLL_ANCHOR_TOLERANCE_PX = 2;
const LOAD_OLDER_THRESHOLD_PX = 32;
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

interface PostSendLock {
  autoFollowOnRelease: boolean;
  baselineContentHeightPx: number;
  messageId: string;
  targetScrollTop: number;
}

export function useChatFeedScroll(options: UseChatFeedScrollOptions) {
  const historySentinelRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLElement>(null);
  const userMessageRowsRef = useRef(new Map<string, HTMLDivElement>());
  const handledPostSendTokenRef = useRef<number | null>(null);
  const pendingPostSendRequestRef = useRef<PostSendFocusRequest | null>(null);
  const postSendLockRef = useRef<PostSendLock | null>(null);
  const postSendLockJustStartedRef = useRef(false);
  const autoFollowRef = useRef(true);
  const olderLoadPendingRef = useRef(false);
  const pendingBottomScrollFrameRef = useRef<number | null>(null);
  const pendingBottomScrollBehaviorRef = useRef<ScrollBehavior>("auto");
  const pendingBottomScrollTokenRef = useRef(0);
  const pendingAnchorScrollFrameRef = useRef<number | null>(null);
  const previousSessionIdRef = useRef<string | undefined>(undefined);
  const trailingSpacerPxRef = useRef(0);
  const userScrollSeenRef = useRef(false);
  const [showScrollDown, setShowScrollDown] = useState(false);
  const [trailingSpacerPx, setTrailingSpacerPxState] = useState(0);
  const [registeredUserRowVersion, setRegisteredUserRowVersion] = useState(0);
  const reply = options.reply;
  const hasReply = Boolean(reply);

  const setTrailingSpacerPx = useCallback((value: number) => {
    const nextValue = Math.max(0, Math.ceil(value));
    trailingSpacerPxRef.current = nextValue;
    setTrailingSpacerPxState(nextValue);
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

    previousSessionIdRef.current = currentSessionId;
    cancelPendingAnchorScroll();
    pendingPostSendRequestRef.current = null;
    postSendLockRef.current = null;
    postSendLockJustStartedRef.current = false;
    olderLoadPendingRef.current = false;
    autoFollowRef.current = true;
    userScrollSeenRef.current = false;
    setTrailingSpacerPx(0);
    setShowScrollDown(false);
    if (scrollRef.current) {
      scrollRef.current.scrollTop = 0;
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

    if (focusUserMessage(request.messageId, "smooth")) {
      handledPostSendTokenRef.current = request.token;
      pendingPostSendRequestRef.current = null;
      return;
    }

    pendingPostSendRequestRef.current = request;
  }, [options.messages, options.postSendFocusRequest, registeredUserRowVersion, setTrailingSpacerPx]);

  useLayoutEffect(() => {
    if (postSendLockRef.current) {
      if (postSendLockJustStartedRef.current) {
        postSendLockJustStartedRef.current = false;
        return;
      }
      syncPostSendLock();
      return;
    }
    if (!options.loadingOlderHistory && autoFollowRef.current && (options.messages.length > 0 || hasReply)) {
      scheduleScrollToBottom(options.statusTone === "loading" ? "auto" : "smooth");
    }
  }, [hasReply, options.loadingOlderHistory, options.messages, options.statusTone, reply]);

  useLayoutEffect(() => {
    if (!options.loadingOlderHistory) {
      olderLoadPendingRef.current = false;
    }
  }, [options.loadingOlderHistory]);

  useLayoutEffect(() => {
    return () => {
      cancelPendingAnchorScroll();
      cancelPendingBottomScroll();
    };
  }, []);

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element || typeof ResizeObserver === "undefined") {
      return;
    }

    const resizeObserver = new ResizeObserver(() => {
      if (postSendLockRef.current) {
        syncPostSendLock();
        return;
      }

      if (!options.loadingOlderHistory && autoFollowRef.current && (options.messages.length > 0 || hasReply)) {
        scheduleScrollToBottom("auto");
        return;
      }

      syncBottomAffordance(element);
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

  useLayoutEffect(() => {
    const root = scrollRef.current;
    const target = historySentinelRef.current;
    if (!root || !target || typeof IntersectionObserver === "undefined") {
      return;
    }

    const observer = new IntersectionObserver((entries) => {
      if (!entries[0]?.isIntersecting || !userScrollSeenRef.current) {
        return;
      }

      maybeLoadOlderHistory(root);
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

    userScrollSeenRef.current = true;
    cancelPendingBottomScroll();
    maybeLoadOlderHistory(element);
    const lock = postSendLockRef.current;
    if (lock) {
      const targetScrollTop = measureUserMessageTargetScrollTop(lock.messageId);
      if (targetScrollTop === null) {
        releasePostSendLock(false);
        return;
      }
      lock.targetScrollTop = targetScrollTop;
      if (Math.abs(element.scrollTop - targetScrollTop) > SCROLL_ANCHOR_TOLERANCE_PX) {
        scrollToAnchor(targetScrollTop, "auto");
      }
      setShowScrollDown(false);
      return;
    }

    if (options.loadingOlderHistory) {
      autoFollowRef.current = false;
      setShowScrollDown(true);
      return;
    }

    syncBottomAffordance(element);
  }

  function resetScrollDown(): void {
    autoFollowRef.current = true;
    userScrollSeenRef.current = false;
    releasePostSendLock(false);
    setShowScrollDown(false);
  }

  function scrollToBottom(behavior: ScrollBehavior = "smooth"): void {
    cancelPendingBottomScroll();
    commitScrollToBottom(behavior);
  }

  function scheduleScrollToBottom(behavior: ScrollBehavior): void {
    autoFollowRef.current = true;
    setShowScrollDown(false);
    pendingBottomScrollBehaviorRef.current = resolvePendingBottomScrollBehavior(
      pendingBottomScrollBehaviorRef.current,
      behavior,
    );
    if (pendingBottomScrollFrameRef.current !== null) {
      return;
    }

    const scheduleToken = pendingBottomScrollTokenRef.current + 1;
    pendingBottomScrollTokenRef.current = scheduleToken;
    pendingBottomScrollFrameRef.current = window.requestAnimationFrame(() => {
      if (pendingBottomScrollTokenRef.current !== scheduleToken) {
        return;
      }
      pendingBottomScrollFrameRef.current = null;
      commitScrollToBottom(pendingBottomScrollBehaviorRef.current);
      pendingBottomScrollBehaviorRef.current = "auto";
    });
  }

  function commitScrollToBottom(behavior: ScrollBehavior): void {
    releasePostSendLock(false);
    autoFollowRef.current = true;
    userScrollSeenRef.current = false;
    setShowScrollDown(false);
    if (scrollRef.current) {
      scrollRef.current.scrollTo({
        top: 0,
        behavior,
      });
    }
  }

  function focusUserMessage(messageId: string, behavior: ScrollBehavior): boolean {
    const targetScrollTop = measureUserMessageTargetScrollTop(messageId);
    const element = scrollRef.current;
    if (targetScrollTop === null || !element) {
      return false;
    }

    const realContentHeightPx = measureRealContentHeight(element);
    postSendLockRef.current = {
      autoFollowOnRelease: true,
      baselineContentHeightPx: realContentHeightPx,
      messageId,
      targetScrollTop,
    };
    postSendLockJustStartedRef.current = true;
    autoFollowRef.current = false;
    setTrailingSpacerPx(requiredTrailingSpacerPx(element, targetScrollTop, realContentHeightPx));
    setShowScrollDown(false);
    scheduleAnchorScroll(() => scrollToAnchor(targetScrollTop, behavior));
    return true;
  }

  function syncPostSendLock(): void {
    const lock = postSendLockRef.current;
    const element = scrollRef.current;
    if (!lock || !element) {
      return;
    }

    const targetScrollTop = measureUserMessageTargetScrollTop(lock.messageId);
    if (targetScrollTop === null) {
      releasePostSendLock(false);
      return;
    }

    lock.targetScrollTop = targetScrollTop;
    const realContentHeightPx = measureRealContentHeight(element);
    const nextSpacerPx = requiredTrailingSpacerPx(element, targetScrollTop, realContentHeightPx);
    if (shouldReleasePostSendLock({
      baselineContentHeightPx: lock.baselineContentHeightPx,
      clientHeight: element.clientHeight,
      hasVisibleContent: hasVisibleContentAfterPostSendAnchor(options.messages, lock.messageId, reply),
      realContentHeightPx,
      targetScrollTop,
    })) {
      releasePostSendLock(lock.autoFollowOnRelease);
      return;
    }

    setTrailingSpacerPx(nextSpacerPx);
    setShowScrollDown(false);
    if (Math.abs(element.scrollTop - targetScrollTop) > SCROLL_ANCHOR_TOLERANCE_PX) {
      scrollToAnchor(targetScrollTop, "auto");
    }
  }

  function releasePostSendLock(restoreBottom: boolean): void {
    if (!postSendLockRef.current && trailingSpacerPxRef.current === 0) {
      return;
    }

    postSendLockRef.current = null;
    postSendLockJustStartedRef.current = false;
    setTrailingSpacerPx(0);
    if (restoreBottom) {
      scheduleScrollToBottom("smooth");
      return;
    }
    if (scrollRef.current) {
      syncBottomAffordance(scrollRef.current);
    }
  }

  function measureUserMessageTargetScrollTop(messageId: string): number | null {
    const element = scrollRef.current;
    const row = userMessageRowsRef.current.get(messageId);
    if (!element || !row) {
      return null;
    }
    return element.scrollTop - (row.getBoundingClientRect().top - element.getBoundingClientRect().top);
  }

  function measureRealContentHeight(element: HTMLElement): number {
    return Math.max(0, element.scrollHeight - trailingSpacerPxRef.current);
  }

  function scrollToAnchor(scrollTop: number, behavior: ScrollBehavior): void {
    scrollRef.current?.scrollTo({ top: scrollTop, behavior });
  }

  function scheduleAnchorScroll(callback: () => void): void {
    cancelPendingAnchorScroll();
    pendingAnchorScrollFrameRef.current = window.requestAnimationFrame(() => {
      pendingAnchorScrollFrameRef.current = null;
      callback();
    });
  }

  function cancelPendingAnchorScroll(): void {
    if (pendingAnchorScrollFrameRef.current === null) {
      return;
    }
    window.cancelAnimationFrame(pendingAnchorScrollFrameRef.current);
    pendingAnchorScrollFrameRef.current = null;
  }

  function cancelPendingBottomScroll(): void {
    if (pendingBottomScrollFrameRef.current === null) {
      return;
    }
    window.cancelAnimationFrame(pendingBottomScrollFrameRef.current);
    pendingBottomScrollFrameRef.current = null;
    pendingBottomScrollBehaviorRef.current = "auto";
    pendingBottomScrollTokenRef.current += 1;
  }

  function maybeLoadOlderHistory(element: HTMLElement): void {
    if (
      !options.hasOlderHistory
      || options.loadingOlderHistory
      || olderLoadPendingRef.current
      || distanceFromHistoryTop(element) > LOAD_OLDER_THRESHOLD_PX
      || !options.onLoadOlderHistory
    ) {
      return;
    }

    olderLoadPendingRef.current = true;
    autoFollowRef.current = false;
    cancelPendingBottomScroll();
    setShowScrollDown(true);
    void options.onLoadOlderHistory().catch(() => {
      olderLoadPendingRef.current = false;
    });
  }

  function syncBottomAffordance(element: HTMLElement): void {
    const shouldShow = shouldShowScrollDown(element);
    autoFollowRef.current = !shouldShow;
    setShowScrollDown(shouldShow);
  }

  return {
    handleScroll,
    historySentinelRef,
    registerUserMessageRow,
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
    trailingSpacerPx,
  };
}

function resolvePendingBottomScrollBehavior(previous: ScrollBehavior, next: ScrollBehavior): ScrollBehavior {
  if (previous === "auto" || next === "auto") {
    return "auto";
  }
  return next;
}

function shouldShowScrollDown(element: HTMLElement): boolean {
  return Math.abs(element.scrollTop) > SCROLL_DOWN_THRESHOLD_PX;
}

function distanceFromHistoryTop(element: HTMLElement): number {
  return element.scrollHeight - element.clientHeight + element.scrollTop;
}

function hasUserMessage(messages: MobileConversationMessage[], messageId: string): boolean {
  return messages.some((message) => message.id === messageId && message.role === "user");
}

function requiredTrailingSpacerPx(element: HTMLElement, targetScrollTop: number, realContentHeightPx: number): number {
  return Math.max(0, Math.abs(targetScrollTop) + element.clientHeight - realContentHeightPx);
}

function shouldReleasePostSendLock(options: {
  baselineContentHeightPx: number;
  clientHeight: number;
  hasVisibleContent: boolean;
  realContentHeightPx: number;
  targetScrollTop: number;
}): boolean {
  return options.hasVisibleContent
    && options.realContentHeightPx > options.baselineContentHeightPx
    && options.realContentHeightPx > Math.abs(options.targetScrollTop) + options.clientHeight;
}

function hasVisibleContentAfterPostSendAnchor(
  messages: MobileConversationMessage[],
  messageId: string,
  reply: AgentPayload | undefined,
): boolean {
  const anchorIndex = messages.findIndex((message) => message.id === messageId);
  if (anchorIndex >= 0 && messages.slice(anchorIndex + 1).some(hasVisibleConversationMessageContent)) {
    return true;
  }
  return Boolean(reply && hasVisibleAgentPayloadContent(reply));
}

function hasVisibleConversationMessageContent(message: MobileConversationMessage): boolean {
  return message.text.trim().length > 0
    || Boolean(message.thinking?.trim())
    || Boolean(message.parts?.length)
    || Boolean(message.tools?.length);
}

function hasVisibleAgentPayloadContent(reply: AgentPayload): boolean {
  return reply.message.trim().length > 0
    || Boolean(reply.thinking?.trim())
    || Boolean(reply.parts?.length)
    || Boolean(reply.tools?.length);
}
