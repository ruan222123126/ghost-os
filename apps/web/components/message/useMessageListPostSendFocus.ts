import type { MutableRefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { StreamingMessageRow } from '@/lib/chat-view/types';
import type { ChatMessage } from '@/lib/types';
import type { PostSendFocusRequest } from '@/hooks/chat/types';

const SCROLL_ANCHOR_TOLERANCE_PX = 2;
type ScrollFrameHandle = number | ReturnType<typeof setTimeout>;

interface PostSendLock {
  autoFollowOnRelease: boolean;
  baselineContentHeightPx: number;
  messageId: string;
  targetScrollTop: number;
}

export interface UseMessageListPostSendFocusOptions {
  autoFollowRef: MutableRefObject<boolean>;
  layoutSignature: string;
  loadingOlderHistory: boolean;
  postSendFocusRequest: PostSendFocusRequest | null;
  scheduleBottomFollow: () => void;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  setShowScrollToBottom: (value: boolean) => void;
  streamingRows: StreamingMessageRow[];
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
    streamingRows,
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

  const measureMessageTargetScrollTop = useCallback((messageId: string): number | null => {
    const container = scrollElementRef.current;
    const row = messageRowsRef.current.get(messageId);
    if (!container || !row) {
      return null;
    }

    return container.scrollTop - (row.getBoundingClientRect().top - container.getBoundingClientRect().top);
  }, [scrollElementRef]);

  const scrollToAnchor = useCallback((scrollTop: number, behavior: ScrollBehavior) => {
    scrollElementRef.current?.scrollTo({ top: scrollTop, behavior });
  }, [scrollElementRef]);

  const scheduleAnchorScroll = useCallback((callback: () => void) => {
    cancelScrollFrame(anchorScrollFrameRef);
    anchorScrollFrameRef.current = requestScrollFrame(() => {
      anchorScrollFrameRef.current = null;
      callback();
    });
  }, []);

  const focusPostSendMessage = useCallback((messageId: string, behavior: ScrollBehavior): boolean => {
    const targetScrollTop = measureMessageTargetScrollTop(messageId);
    const container = scrollElementRef.current;
    if (targetScrollTop === null || !container) {
      return false;
    }

    const realContentHeightPx = measureRealContentHeight(container, trailingSpacerPxRef);
    postSendLockRef.current = {
      autoFollowOnRelease: true,
      baselineContentHeightPx: realContentHeightPx,
      messageId,
      targetScrollTop,
    };
    postSendLockJustStartedRef.current = true;
    autoFollowRef.current = false;
    setTrailingSpacerPx(requiredTrailingSpacerPx(container, targetScrollTop, realContentHeightPx));
    setShowScrollToBottom(false);
    scheduleAnchorScroll(() => {
      postSendLockJustStartedRef.current = false;
      scrollToAnchor(targetScrollTop, behavior);
    });
    return true;
  }, [
    autoFollowRef,
    measureMessageTargetScrollTop,
    scheduleAnchorScroll,
    scrollToAnchor,
    scrollElementRef,
    setShowScrollToBottom,
    setTrailingSpacerPx,
  ]);

  const syncPostSendLock = useCallback(() => {
    const lock = postSendLockRef.current;
    const container = scrollElementRef.current;
    if (!lock || !container) {
      return;
    }

    const targetScrollTop = measureMessageTargetScrollTop(lock.messageId);
    if (targetScrollTop === null) {
      releasePostSendLock(false);
      return;
    }

    lock.targetScrollTop = targetScrollTop;
    const realContentHeightPx = measureRealContentHeight(container, trailingSpacerPxRef);
    if (shouldReleasePostSendLock({
      baselineContentHeightPx: lock.baselineContentHeightPx,
      clientHeight: container.clientHeight,
      hasVisibleContent: hasVisibleContentAfterPostSendAnchor(
        visibleCommittedMessages,
        lock.messageId,
        streamingRows,
      ),
      realContentHeightPx,
      targetScrollTop,
    })) {
      releasePostSendLock(lock.autoFollowOnRelease);
      return;
    }

    setTrailingSpacerPx(requiredTrailingSpacerPx(container, targetScrollTop, realContentHeightPx));
    setShowScrollToBottom(false);
    if (Math.abs(container.scrollTop - targetScrollTop) > SCROLL_ANCHOR_TOLERANCE_PX) {
      scrollToAnchor(targetScrollTop, 'auto');
    }
  }, [
    measureMessageTargetScrollTop,
    releasePostSendLock,
    scrollToAnchor,
    scrollElementRef,
    setShowScrollToBottom,
    setTrailingSpacerPx,
    streamingRows,
    visibleCommittedMessages,
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

    if (focusPostSendMessage(request.messageId, 'smooth')) {
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
    if (loadingOlderHistory || !postSendLockRef.current) {
      return;
    }
    if (!postSendLockJustStartedRef.current) {
      syncPostSendLock();
    }
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
  container: HTMLElement,
  targetScrollTop: number,
  realContentHeightPx: number,
): number {
  return Math.max(0, Math.abs(targetScrollTop) + container.clientHeight - realContentHeightPx);
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

function hasUserMessage(messages: ChatMessage[], messageId: string): boolean {
  return messages.some((message) => message.id === messageId && message.kind === 'user');
}

function hasVisibleContentAfterPostSendAnchor(
  messages: ChatMessage[],
  messageId: string,
  streamingRows: StreamingMessageRow[],
): boolean {
  const anchorIndex = messages.findIndex((message) => message.id === messageId);
  if (anchorIndex >= 0 && messages.slice(anchorIndex + 1).some(hasVisibleMessageContent)) {
    return true;
  }

  return streamingRows.some((row) => hasVisibleMessageContent(row.message));
}

function hasVisibleMessageContent(message: ChatMessage): boolean {
  return message.content.trim().length > 0;
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
