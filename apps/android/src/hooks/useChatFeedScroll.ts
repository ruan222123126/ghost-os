import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import type { AgentPayload, MobileConversationMessage, StatusMessage } from "../mobileTypes";

const SCROLL_DOWN_THRESHOLD_PX = 50;
const SCROLL_ANCHOR_TOLERANCE_PX = 2;
const LOAD_OLDER_THRESHOLD_PX = 32;

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
  anchorTop: number;
  anchored: boolean;
  autoFollowOnRelease: boolean;
  baselineContentHeightPx: number;
  messageId: string;
}

export function useChatFeedScroll(options: UseChatFeedScrollOptions) {
  const scrollRef = useRef<HTMLElement>(null);
  const userMessageRowsRef = useRef(new Map<string, HTMLDivElement>());
  const handledPostSendTokenRef = useRef<number | null>(null);
  const pendingPostSendRequestRef = useRef<PostSendFocusRequest | null>(null);
  const postSendLockRef = useRef<PostSendLock | null>(null);
  const postSendLockJustStartedRef = useRef(false);
  const autoFollowRef = useRef(true);
  const olderLoadPendingRef = useRef(false);
  const olderLoadAnchorRef = useRef<OlderLoadAnchor | null>(null);
  const previousSessionIdRef = useRef<string | undefined>(undefined);
  const forceBottomOnNextContentRef = useRef(false);
  const trailingSpacerPxRef = useRef(0);
  const animationFrameRef = useRef<number | null>(null);
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
    cancelScheduledScroll();
    pendingPostSendRequestRef.current = null;
    postSendLockRef.current = null;
    postSendLockJustStartedRef.current = false;
    olderLoadPendingRef.current = false;
    olderLoadAnchorRef.current = null;
    autoFollowRef.current = true;
    forceBottomOnNextContentRef.current = !options.postSendFocusRequest;
    setTrailingSpacerPx(0);
    setShowScrollDown(false);
    if (scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [options.postSendFocusRequest, options.sessionId, setTrailingSpacerPx]);

  useLayoutEffect(() => {
    if (restoreOlderLoadAnchor()) {
      return;
    }
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
  }, [options.messages, options.postSendFocusRequest, registeredUserRowVersion]);

  useLayoutEffect(() => {
    if (restoreOlderLoadAnchor()) {
      return;
    }
    if (postSendLockRef.current) {
      if (postSendLockJustStartedRef.current) {
        postSendLockJustStartedRef.current = false;
        return;
      }
      syncPostSendLock();
      return;
    }
    if (autoFollowRef.current && (options.messages.length > 0 || hasReply)) {
      scrollToBottom(forceBottomOnNextContentRef.current ? "auto" : "smooth");
      forceBottomOnNextContentRef.current = false;
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
  }, [hasReply, options.messages.length, options.statusTone, reply]);

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
      maybeLoadOlderHistory(element);
      if (olderLoadAnchorRef.current || options.loadingOlderHistory) {
        autoFollowRef.current = false;
        setShowScrollDown(false);
        return;
      }
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

    const realContentHeightPx = measureRealContentHeight(element);
    postSendLockRef.current = {
      anchorTop,
      anchored: false,
      autoFollowOnRelease: true,
      baselineContentHeightPx: realContentHeightPx,
      messageId,
    };
    postSendLockJustStartedRef.current = true;
    forceBottomOnNextContentRef.current = false;
    setTrailingSpacerPx(requiredTrailingSpacerPx(element, anchorTop, realContentHeightPx));
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
    const realContentHeightPx = measureRealContentHeight(element);
    const nextSpacerPx = requiredTrailingSpacerPx(element, anchorTop, realContentHeightPx);
    if (shouldReleasePostSendLock({
      anchorTop,
      baselineContentHeightPx: lock.baselineContentHeightPx,
      clientHeight: element.clientHeight,
      hasVisibleContent: hasVisibleContentAfterPostSendAnchor(options.messages, lock.messageId, reply),
      realContentHeightPx,
    })) {
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

  function measureRealContentHeight(element: HTMLElement): number {
    return Math.max(0, element.scrollHeight - trailingSpacerPxRef.current);
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
    scrollToAnchor(anchor.scrollTop + delta, "auto");
    setShowScrollDown(shouldShowScrollDown(element));
    return true;
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

interface OlderLoadAnchor {
  scrollHeight: number;
  scrollTop: number;
}

function shouldShowScrollDown(element: HTMLElement): boolean {
  return element.scrollHeight - element.scrollTop - element.clientHeight > SCROLL_DOWN_THRESHOLD_PX;
}

function hasUserMessage(messages: MobileConversationMessage[], messageId: string): boolean {
  return messages.some((message) => message.id === messageId && message.role === "user");
}

function requiredTrailingSpacerPx(element: HTMLElement, anchorTop: number, realContentHeightPx: number): number {
  return Math.max(0, anchorTop + element.clientHeight - realContentHeightPx);
}

function shouldReleasePostSendLock(options: {
  anchorTop: number;
  baselineContentHeightPx: number;
  clientHeight: number;
  hasVisibleContent: boolean;
  realContentHeightPx: number;
}): boolean {
  return options.hasVisibleContent
    && options.realContentHeightPx > options.baselineContentHeightPx
    && options.realContentHeightPx > options.anchorTop + options.clientHeight;
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
  return hasVisibleReplyContent(reply);
}

function hasVisibleConversationMessageContent(message: MobileConversationMessage): boolean {
  if (message.role === "user") {
    return false;
  }
  return hasVisibleAssistantContent({
    message: message.text,
    parts: message.parts,
    thinking: message.thinking,
    tools: message.tools,
  });
}

function hasVisibleReplyContent(reply: AgentPayload | undefined): boolean {
  if (!reply) {
    return false;
  }
  return hasVisibleAssistantContent(reply);
}

function hasVisibleAssistantContent(input: Pick<AgentPayload, "message" | "parts" | "thinking" | "tools">): boolean {
  return hasNonBlankText(input.message)
    || hasNonBlankText(input.thinking)
    || Boolean(input.parts?.some(hasVisibleAssistantPart))
    || Boolean(input.tools?.some(hasVisibleToolContent));
}

function hasVisibleAssistantPart(part: NonNullable<AgentPayload["parts"]>[number]): boolean {
  if (part.kind === "text") {
    return hasNonBlankText(part.text);
  }
  return hasVisibleToolContent(part.tool);
}

function hasVisibleToolContent(tool: NonNullable<AgentPayload["tools"]>[number]): boolean {
  return hasNonBlankText(tool.toolName)
    || hasNonBlankText(tool.input)
    || hasNonBlankText(tool.output)
    || hasNonBlankText(tool.error)
    || hasNonBlankText(tool.approvalId);
}

function hasNonBlankText(value: string | undefined): boolean {
  return Boolean(value?.trim());
}
