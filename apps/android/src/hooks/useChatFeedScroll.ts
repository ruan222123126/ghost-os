import { useLayoutEffect, useRef, useState } from "react";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "../mobileTypes";

const SCROLL_DOWN_THRESHOLD_PX = 50;
const LOAD_OLDER_THRESHOLD_PX = 32;

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
  scrollHeight: number;
  scrollTop: number;
}

export function useChatFeedScroll(options: UseChatFeedScrollOptions) {
  const scrollRef = useRef<HTMLElement>(null);
  const handledPostSendTokenRef = useRef<number | null>(null);
  const autoFollowRef = useRef(true);
  const olderLoadPendingRef = useRef(false);
  const olderLoadAnchorRef = useRef<OlderLoadAnchor | null>(null);
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
    olderLoadAnchorRef.current = null;
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
      olderLoadAnchorRef.current = null;
    }
  }, [options.loadingOlderHistory]);

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element || typeof ResizeObserver === "undefined") {
      return;
    }

    const resizeObserver = new ResizeObserver(() => {
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
  }, [hasReply, options.messages.length, options.statusTone, reply]);

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

  function scrollToAnchor(anchorTop: number, behavior: ScrollBehavior): void {
    scrollRef.current?.scrollTo({ top: anchorTop, behavior });
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
      scrollHeight: element.scrollHeight,
      scrollTop: element.scrollTop,
    };
    autoFollowRef.current = false;
    void options.onLoadOlderHistory().catch(() => {
      olderLoadPendingRef.current = false;
      olderLoadAnchorRef.current = null;
    });
  }

  function restoreOlderLoadAnchor(): boolean {
    const anchor = olderLoadAnchorRef.current;
    const element = scrollRef.current;
    if (!anchor || !element || element.scrollHeight <= anchor.scrollHeight) {
      return false;
    }

    const delta = element.scrollHeight - anchor.scrollHeight;
    olderLoadAnchorRef.current = null;
    autoFollowRef.current = false;
    scrollToAnchor(anchor.scrollTop + delta, "auto");
    setShowScrollDown(shouldShowScrollDown(element));
    return true;
  }

  return {
    handleScroll,
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
  };
}

function shouldShowScrollDown(element: HTMLElement): boolean {
  return element.scrollHeight - element.scrollTop - element.clientHeight > SCROLL_DOWN_THRESHOLD_PX;
}
