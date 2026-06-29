import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import type { MobileConversationMessage, StatusMessage } from "../mobileTypes";

const SCROLL_DOWN_THRESHOLD_PX = 50;
const SCROLL_ANCHOR_TOLERANCE_PX = 2;

interface UseChatFeedScrollOptions {
  messages: MobileConversationMessage[];
  postSendFocusRequest?: PostSendFocusRequest | null;
  reply: unknown;
  statusTone: StatusMessage["tone"];
}

interface PostSendFocusRequest {
  messageId: string;
  token: number;
}

interface PostSendLock {
  anchorTop: number;
  anchored: boolean;
  autoFollowOnRelease: boolean;
  messageId: string;
}

export function useChatFeedScroll(options: UseChatFeedScrollOptions) {
  const scrollRef = useRef<HTMLElement>(null);
  const userMessageRowsRef = useRef(new Map<string, HTMLDivElement>());
  const handledPostSendTokenRef = useRef<number | null>(null);
  const postSendLockRef = useRef<PostSendLock | null>(null);
  const postSendLockJustStartedRef = useRef(false);
  const autoFollowRef = useRef(true);
  const trailingSpacerPxRef = useRef(0);
  const animationFrameRef = useRef<number | null>(null);
  const [showScrollDown, setShowScrollDown] = useState(false);
  const [trailingSpacerPx, setTrailingSpacerPxState] = useState(0);
  const hasReply = Boolean(options.reply);

  const setTrailingSpacerPx = useCallback((value: number) => {
    const nextValue = Math.max(0, Math.ceil(value));
    trailingSpacerPxRef.current = nextValue;
    setTrailingSpacerPxState(nextValue);
  }, []);

  const registerUserMessageRow = useCallback((messageId: string) => {
    return (node: HTMLDivElement | null) => {
      if (node) {
        userMessageRowsRef.current.set(messageId, node);
        return;
      }
      userMessageRowsRef.current.delete(messageId);
    };
  }, []);

  useLayoutEffect(() => {
    const request = options.postSendFocusRequest;
    if (!request || handledPostSendTokenRef.current === request.token) {
      return;
    }
    if (!hasUserMessage(options.messages, request.messageId)) {
      return;
    }
    if (focusUserMessage(request.messageId, "smooth")) {
      handledPostSendTokenRef.current = request.token;
    }
  }, [options.messages, options.postSendFocusRequest]);

  useLayoutEffect(() => {
    if (postSendLockRef.current) {
      if (postSendLockJustStartedRef.current) {
        postSendLockJustStartedRef.current = false;
        return;
      }
      syncPostSendLock();
      return;
    }
    if (autoFollowRef.current && (options.messages.length > 0 || hasReply)) {
      scrollToBottom("smooth");
    }
  }, [hasReply, options.messages, options.statusTone]);

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

      if (autoFollowRef.current && (options.messages.length > 0 || hasReply)) {
        scrollToBottom("auto");
        return;
      }

      const shouldShow = shouldShowScrollDown(element);
      autoFollowRef.current = !shouldShow;
      setShowScrollDown(shouldShow);
    });

    resizeObserver.observe(element);
    return () => {
      resizeObserver.disconnect();
    };
  }, [hasReply, options.messages.length, options.statusTone]);

  useEffect(() => {
    return () => {
      cancelScheduledScroll();
    };
  }, []);

  function handleScroll(): void {
    const element = scrollRef.current;
    if (!element) {
      return;
    }

    const lock = postSendLockRef.current;
    if (!lock) {
      const shouldShow = shouldShowScrollDown(element);
      autoFollowRef.current = !shouldShow;
      setShowScrollDown(shouldShow);
      return;
    }

    if (!lock.anchored && Math.abs(element.scrollTop - lock.anchorTop) <= SCROLL_ANCHOR_TOLERANCE_PX) {
      lock.anchored = true;
    }
    if (lock.anchored && element.scrollTop < lock.anchorTop - SCROLL_ANCHOR_TOLERANCE_PX) {
      cancelPostSendLock();
      return;
    }
    if (element.scrollTop > lock.anchorTop + SCROLL_ANCHOR_TOLERANCE_PX) {
      scrollToAnchor(lock.anchorTop, "auto");
    }
    setShowScrollDown(false);
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

  function cancelPostSendLock(): void {
    postSendLockRef.current = null;
    postSendLockJustStartedRef.current = false;
    autoFollowRef.current = false;
    setTrailingSpacerPx(0);
    if (scrollRef.current) {
      setShowScrollDown(shouldShowScrollDown(scrollRef.current));
    }
  }

  function releasePostSendLock(restoreBottom: boolean): void {
    postSendLockRef.current = null;
    postSendLockJustStartedRef.current = false;
    setTrailingSpacerPx(0);
    if (restoreBottom) {
      scheduleScroll(() => scrollToBottom("smooth"));
      return;
    }
    if (scrollRef.current) {
      const shouldShow = shouldShowScrollDown(scrollRef.current);
      autoFollowRef.current = !shouldShow;
      setShowScrollDown(shouldShow);
    }
  }

  function focusUserMessage(messageId: string, behavior: ScrollBehavior): boolean {
    const anchorTop = measureUserMessageAnchorTop(messageId);
    const element = scrollRef.current;
    if (anchorTop === null || !element) {
      return false;
    }

    postSendLockRef.current = {
      anchorTop,
      anchored: false,
      autoFollowOnRelease: true,
      messageId,
    };
    postSendLockJustStartedRef.current = true;
    setTrailingSpacerPx(requiredTrailingSpacerPx(element, anchorTop, trailingSpacerPxRef.current));
    setShowScrollDown(false);
    scheduleScroll(() => scrollToAnchor(anchorTop, behavior));
    return true;
  }

  function syncPostSendLock(): void {
    const lock = postSendLockRef.current;
    const element = scrollRef.current;
    if (!lock || !element) {
      return;
    }

    const anchorTop = measureUserMessageAnchorTop(lock.messageId);
    if (anchorTop === null) {
      releasePostSendLock(false);
      return;
    }

    lock.anchorTop = anchorTop;
    const nextSpacerPx = requiredTrailingSpacerPx(element, anchorTop, trailingSpacerPxRef.current);
    if (nextSpacerPx <= 0) {
      releasePostSendLock(lock.autoFollowOnRelease);
      return;
    }

    setTrailingSpacerPx(nextSpacerPx);
    setShowScrollDown(false);
    if (element.scrollTop > anchorTop + SCROLL_ANCHOR_TOLERANCE_PX) {
      scrollToAnchor(anchorTop, "auto");
    }
  }

  function measureUserMessageAnchorTop(messageId: string): number | null {
    const element = scrollRef.current;
    const row = userMessageRowsRef.current.get(messageId);
    if (!element || !row) {
      return null;
    }
    return row.getBoundingClientRect().top - element.getBoundingClientRect().top + element.scrollTop;
  }

  function scrollToAnchor(anchorTop: number, behavior: ScrollBehavior): void {
    scrollRef.current?.scrollTo({ top: anchorTop, behavior });
  }

  function scheduleScroll(callback: () => void): void {
    cancelScheduledScroll();
    if (typeof window === "undefined" || typeof window.requestAnimationFrame !== "function") {
      callback();
      return;
    }
    animationFrameRef.current = window.requestAnimationFrame(() => {
      animationFrameRef.current = null;
      callback();
    });
  }

  function cancelScheduledScroll(): void {
    if (
      animationFrameRef.current === null
      || typeof window === "undefined"
      || typeof window.cancelAnimationFrame !== "function"
    ) {
      animationFrameRef.current = null;
      return;
    }
    window.cancelAnimationFrame(animationFrameRef.current);
    animationFrameRef.current = null;
  }

  return {
    handleScroll,
    registerUserMessageRow,
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
    trailingSpacerPx,
  };
}

function shouldShowScrollDown(element: HTMLElement): boolean {
  return element.scrollHeight - element.scrollTop - element.clientHeight > SCROLL_DOWN_THRESHOLD_PX;
}

function hasUserMessage(messages: MobileConversationMessage[], messageId: string): boolean {
  return messages.some((message) => message.id === messageId && message.role === "user");
}

function requiredTrailingSpacerPx(element: HTMLElement, anchorTop: number, currentSpacerPx: number): number {
  const baseScrollHeight = Math.max(0, element.scrollHeight - currentSpacerPx);
  return Math.max(0, anchorTop + element.clientHeight - baseScrollHeight);
}
