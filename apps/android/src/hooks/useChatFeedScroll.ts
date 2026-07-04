import { useLayoutEffect, useRef, useState } from "react";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "../mobileTypes";

const SCROLL_DOWN_THRESHOLD_PX = 50;
const LOAD_OLDER_THRESHOLD_PX = 32;
const CHAT_FEED_CONTENT_SELECTOR = "[data-chat-feed-content]";

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

export function useChatFeedScroll(options: UseChatFeedScrollOptions) {
  const historySentinelRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLElement>(null);
  const handledPostSendTokenRef = useRef<number | null>(null);
  const autoFollowRef = useRef(true);
  const olderLoadPendingRef = useRef(false);
  const pendingBottomScrollFrameRef = useRef<number | null>(null);
  const pendingBottomScrollBehaviorRef = useRef<ScrollBehavior>("auto");
  const pendingBottomScrollTokenRef = useRef(0);
  const previousSessionIdRef = useRef<string | undefined>(undefined);
  const userScrollSeenRef = useRef(false);
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
    autoFollowRef.current = true;
    userScrollSeenRef.current = false;
    setShowScrollDown(false);
    if (scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [options.sessionId]);

  useLayoutEffect(() => {
    const request = options.postSendScrollRequest;
    if (!request || handledPostSendTokenRef.current === request.token) {
      return;
    }

    handledPostSendTokenRef.current = request.token;
    scrollToBottom("auto");
  }, [options.messages, options.postSendScrollRequest]);

  useLayoutEffect(() => {
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
      cancelPendingBottomScroll();
    };
  }, []);

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element || typeof ResizeObserver === "undefined") {
      return;
    }

    const resizeObserver = new ResizeObserver(() => {
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
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
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
