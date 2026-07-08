import type { MutableRefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { ChatMessage } from '@/lib/types';
import type { PostSendFocusRequest } from '@/hooks/chat/types';

const POST_SEND_ANCHOR_TOP_OFFSET_PX = 32;
const SCROLL_ANCHOR_TOLERANCE_PX = 2;
type ScrollFrameHandle = number | ReturnType<typeof setTimeout>;

interface PostSendLock {
  anchorTopOffsetPx: number;
  messageId: string;
  programmaticScrollTarget: number | null;
  reservedViewportBottomScrollTop: number;
  targetScrollTop: number;
}

interface AnchorMeasurement {
  targetScrollTop: number;
  topPx: number;
}

export interface UseMessageListPostSendFocusOptions {
  autoFollowRef: MutableRefObject<boolean>;
  layoutSignature: string;
  loadingOlderHistory: boolean;
  postSendFocusRequest: PostSendFocusRequest | null;
  scheduleBottomFollow: () => void;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  setShowScrollToBottom: (value: boolean) => void;
  syncCurrentBottomAffordance: () => void;
  visibleCommittedMessages: ChatMessage[];
}

export function useMessageListPostSendFocus(options: UseMessageListPostSendFocusOptions) {
  const {
    autoFollowRef,
    layoutSignature,
    loadingOlderHistory,
    postSendFocusRequest,
    scheduleBottomFollow,
    scrollElementRef,
    setShowScrollToBottom,
    syncCurrentBottomAffordance,
    visibleCommittedMessages,
  } = options;
  const anchorScrollFrameRef = useRef<ScrollFrameHandle | null>(null);
  const messageRowsRef = useRef(new Map<string, HTMLDivElement>());
  const handledPostSendTokenRef = useRef<number | null>(null);
  const pendingPostSendRequestRef = useRef<PostSendFocusRequest | null>(null);
  const postSendLockRef = useRef<PostSendLock | null>(null);
  const postSendLockJustStartedRef = useRef(false);
  const trailingSpacerPxRef = useRef(0);
  const [registeredMessageRowVersion, setRegisteredMessageRowVersion] = useState(0);
  const [trailingSpacerPx, setTrailingSpacerPxState] = useState(0);

  const setTrailingSpacerPx = useCallback((value: number) => {
    const nextValue = Math.max(0, Math.ceil(value));
    if (trailingSpacerPxRef.current === nextValue) {
      return;
    }

    trailingSpacerPxRef.current = nextValue;
    setTrailingSpacerPxState(nextValue);
  }, []);

  const registerMessageRow = useCallback((messageId: string) => {
    return (node: HTMLDivElement | null) => {
      if (node) {
        messageRowsRef.current.set(messageId, node);
        if (pendingPostSendRequestRef.current?.messageId === messageId) {
          setRegisteredMessageRowVersion((version) => version + 1);
        }
        return;
      }

      messageRowsRef.current.delete(messageId);
    };
  }, []);

  const cancelAnchorScroll = useCallback(() => {
    cancelScrollFrame(anchorScrollFrameRef);
  }, []);

  const measureMessageAnchor = useCallback((
    messageId: string,
    anchorTopOffsetPx = POST_SEND_ANCHOR_TOP_OFFSET_PX,
  ): AnchorMeasurement | null => {
    const container = scrollElementRef.current;
    const row = messageRowsRef.current.get(messageId);
    if (!container || !row) {
      return null;
    }

    const topPx = row.getBoundingClientRect().top - container.getBoundingClientRect().top;
    return {
      targetScrollTop: Math.max(0, container.scrollTop + topPx - anchorTopOffsetPx),
      topPx,
    };
  }, [scrollElementRef]);

  const scrollToAnchor = useCallback((scrollTop: number, behavior: ScrollBehavior) => {
    scrollElementRef.current?.scrollTo({ top: scrollTop, behavior });
  }, [scrollElementRef]);

  const releasePostSendLock = useCallback((restoreBottom: boolean) => {
    if (!postSendLockRef.current && trailingSpacerPxRef.current === 0) {
      return;
    }

    postSendLockRef.current = null;
    postSendLockJustStartedRef.current = false;
    setTrailingSpacerPx(0);
    if (restoreBottom) {
      autoFollowRef.current = true;
      scheduleBottomFollow();
      return;
    }

    syncCurrentBottomAffordance();
  }, [autoFollowRef, scheduleBottomFollow, setTrailingSpacerPx, syncCurrentBottomAffordance]);

  const scheduleAnchorScroll = useCallback((callback: () => void) => {
    cancelScrollFrame(anchorScrollFrameRef);
    anchorScrollFrameRef.current = requestScrollFrame(() => {
      anchorScrollFrameRef.current = null;
      callback();
    });
  }, []);

  const focusPostSendMessage = useCallback((messageId: string, behavior: ScrollBehavior): boolean => {
    const anchor = measureMessageAnchor(messageId);
    const container = scrollElementRef.current;
    if (!anchor || !container) {
      return false;
    }

    const realContentHeightPx = measureRealContentHeight(container, trailingSpacerPxRef);
    const reservedViewportBottomScrollTop = anchor.targetScrollTop + container.clientHeight;
    const trailingSpacer = requiredTrailingSpacerPx(realContentHeightPx, reservedViewportBottomScrollTop);
    postSendLockRef.current = {
      anchorTopOffsetPx: POST_SEND_ANCHOR_TOP_OFFSET_PX,
      messageId,
      programmaticScrollTarget: anchor.targetScrollTop,
      reservedViewportBottomScrollTop,
      targetScrollTop: anchor.targetScrollTop,
    };
    postSendLockJustStartedRef.current = true;
    autoFollowRef.current = false;
    setTrailingSpacerPx(trailingSpacer);
    setShowScrollToBottom(false);
    scheduleAnchorScroll(() => {
      postSendLockJustStartedRef.current = false;
      scrollToAnchor(anchor.targetScrollTop, behavior);
    });
    return true;
  }, [
    autoFollowRef,
    measureMessageAnchor,
    scheduleAnchorScroll,
    scrollElementRef,
    scrollToAnchor,
    setShowScrollToBottom,
    setTrailingSpacerPx,
  ]);

  const syncPostSendLock = useCallback(() => {
    const lock = postSendLockRef.current;
    const container = scrollElementRef.current;
    if (!lock || !container) {
      return;
    }
    if (isProgrammaticScrollInProgress(container, lock)) {
      setShowScrollToBottom(false);
      return;
    }
    lock.programmaticScrollTarget = null;

    if (!messageRowsRef.current.has(lock.messageId)) {
      releasePostSendLock(false);
      return;
    }

    const anchor = measureMessageAnchor(lock.messageId, lock.anchorTopOffsetPx);
    if (!anchor) {
      releasePostSendLock(false);
      return;
    }

    const realContentHeightPx = measureRealContentHeight(container, trailingSpacerPxRef);
    const spacerForLockedTarget = requiredTrailingSpacerPx(
      realContentHeightPx,
      lock.reservedViewportBottomScrollTop,
    );
    if (spacerForLockedTarget > 0) {
      setTrailingSpacerPx(spacerForLockedTarget);
      setShowScrollToBottom(false);
      if (Math.abs(container.scrollTop - lock.targetScrollTop) > SCROLL_ANCHOR_TOLERANCE_PX) {
        lock.programmaticScrollTarget = lock.targetScrollTop;
        scrollToAnchor(lock.targetScrollTop, 'auto');
      }
      return;
    }

    lock.targetScrollTop = anchor.targetScrollTop;
    setTrailingSpacerPx(0);
    setShowScrollToBottom(false);
    if (Math.abs(container.scrollTop - anchor.targetScrollTop) > SCROLL_ANCHOR_TOLERANCE_PX) {
      lock.programmaticScrollTarget = anchor.targetScrollTop;
      scrollToAnchor(anchor.targetScrollTop, 'auto');
    }
  }, [
    measureMessageAnchor,
    releasePostSendLock,
    scrollElementRef,
    scrollToAnchor,
    setShowScrollToBottom,
    setTrailingSpacerPx,
  ]);

  usePostSendFocusRequest({
    focusPostSendMessage,
    handledPostSendTokenRef,
    pendingPostSendRequestRef,
    postSendFocusRequest,
    registeredMessageRowVersion,
    visibleCommittedMessages,
  });
  usePostSendLockSync({
    layoutSignature,
    loadingOlderHistory,
    postSendLockJustStartedRef,
    postSendLockRef,
    syncPostSendLock,
  });
  useEffect(() => {
    return () => {
      cancelScrollFrame(anchorScrollFrameRef);
    };
  }, []);

  return {
    cancelAnchorScroll,
    postSendLockJustStartedRef,
    postSendLockRef,
    registerMessageRow,
    releasePostSendLock,
    syncPostSendLock,
    trailingSpacerPx,
  };
}

function usePostSendFocusRequest(options: {
  focusPostSendMessage: (messageId: string, behavior: ScrollBehavior) => boolean;
  handledPostSendTokenRef: MutableRefObject<number | null>;
  pendingPostSendRequestRef: MutableRefObject<PostSendFocusRequest | null>;
  postSendFocusRequest: PostSendFocusRequest | null;
  registeredMessageRowVersion: number;
  visibleCommittedMessages: ChatMessage[];
}) {
  const {
    focusPostSendMessage,
    handledPostSendTokenRef,
    pendingPostSendRequestRef,
    postSendFocusRequest,
    registeredMessageRowVersion,
    visibleCommittedMessages,
  } = options;

  useLayoutEffect(() => {
    const request = postSendFocusRequest;
    if (!request || handledPostSendTokenRef.current === request.token) {
      pendingPostSendRequestRef.current = null;
      return;
    }
    if (!hasUserMessage(visibleCommittedMessages, request.messageId)) {
      pendingPostSendRequestRef.current = null;
      return;
    }

    if (focusPostSendMessage(request.messageId, 'auto')) {
      handledPostSendTokenRef.current = request.token;
      pendingPostSendRequestRef.current = null;
      return;
    }

    pendingPostSendRequestRef.current = request;
  }, [
    focusPostSendMessage,
    handledPostSendTokenRef,
    pendingPostSendRequestRef,
    postSendFocusRequest,
    registeredMessageRowVersion,
    visibleCommittedMessages,
  ]);
}

function usePostSendLockSync(options: {
  layoutSignature: string;
  loadingOlderHistory: boolean;
  postSendLockJustStartedRef: MutableRefObject<boolean>;
  postSendLockRef: MutableRefObject<PostSendLock | null>;
  syncPostSendLock: () => void;
}) {
  const {
    layoutSignature,
    loadingOlderHistory,
    postSendLockJustStartedRef,
    postSendLockRef,
    syncPostSendLock,
  } = options;

  useLayoutEffect(() => {
    if (loadingOlderHistory || !postSendLockRef.current || postSendLockJustStartedRef.current) {
      return;
    }
    syncPostSendLock();
  }, [
    layoutSignature,
    loadingOlderHistory,
    postSendLockJustStartedRef,
    postSendLockRef,
    syncPostSendLock,
  ]);
}

function measureRealContentHeight(
  container: HTMLElement,
  trailingSpacerPxRef: MutableRefObject<number>,
): number {
  return Math.max(0, container.scrollHeight - trailingSpacerPxRef.current);
}

function requiredTrailingSpacerPx(
  realContentHeightPx: number,
  reservedViewportBottomScrollTop: number,
): number {
  return Math.max(0, reservedViewportBottomScrollTop - realContentHeightPx);
}

function isProgrammaticScrollInProgress(container: HTMLElement, lock: PostSendLock): boolean {
  if (lock.programmaticScrollTarget === null) {
    return false;
  }

  return Math.abs(container.scrollTop - lock.programmaticScrollTarget) > SCROLL_ANCHOR_TOLERANCE_PX;
}

function hasUserMessage(messages: ChatMessage[], messageId: string): boolean {
  return messages.some((message) => message.id === messageId && message.kind === 'user');
}

function requestScrollFrame(callback: FrameRequestCallback): ScrollFrameHandle {
  if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
    return window.requestAnimationFrame(callback);
  }

  return setTimeout(() => callback(Date.now()), 16);
}

function cancelScrollFrame(scrollFrameRef: MutableRefObject<ScrollFrameHandle | null>) {
  if (scrollFrameRef.current === null) {
    return;
  }

  if (
    typeof window !== 'undefined'
    && typeof window.cancelAnimationFrame === 'function'
    && typeof scrollFrameRef.current === 'number'
  ) {
    window.cancelAnimationFrame(scrollFrameRef.current);
  } else {
    clearTimeout(scrollFrameRef.current);
  }
  scrollFrameRef.current = null;
}
