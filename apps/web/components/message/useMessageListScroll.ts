import type { MutableRefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { ChatMessage } from '@/lib/types';
import type { PostSendFocusRequest } from '@/hooks/chat/types';
import { resolveMessageListAutoFollow } from './messageListScroll';
import { useMessageListPostSendFocus } from './useMessageListPostSendFocus';

const OLDER_HISTORY_TOP_THRESHOLD_PX = 240;
const HARD_BOTTOM_TOLERANCE_PX = 2;
type ScrollFrameHandle = number | ReturnType<typeof setTimeout>;

interface OlderHistoryAnchorSnapshot {
  scrollHeight: number;
  scrollTop: number;
}

export interface UseMessageListScrollOptions {
  hasOlderHistory: boolean;
  loadingOlderHistory: boolean;
  loadOlderHistory: () => Promise<void>;
  layoutSignature: string;
  postSendFocusRequest?: PostSendFocusRequest | null;
  rowCount: number;
  visibleCommittedMessages: ChatMessage[];
}

export function useMessageListScroll(options: UseMessageListScrollOptions) {
  const historySentinelRef = useRef<HTMLDivElement>(null);
  const scrollElementRef = useRef<HTMLDivElement>(null);
  const [olderHistoryLoadingPaused] = useState(false);
  const [showScrollToBottom, setShowScrollToBottom] = useState(false);
  const autoFollowRef = useRef(true);
  const mountedRef = useRef(false);
  const olderLoadPendingRef = useRef(false);
  const olderHistoryAnchorRef = useRef<OlderHistoryAnchorSnapshot | null>(null);
  const scrollFrameRef = useRef<ScrollFrameHandle | null>(null);
  const skipOlderHistoryLoadOnceRef = useRef(false);

  const scheduleBottomFollow = useCallback(() => {
    scheduleScrollToBottom({
      autoFollowRef,
      scrollElementRef,
      scrollFrameRef,
    });
  }, []);

  const syncCurrentBottomAffordance = useCallback(() => {
    const container = scrollElementRef.current;
    if (container) {
      syncBottomAffordance(container, autoFollowRef, setShowScrollToBottom);
    }
  }, []);

  useInitialBottomPlacement({
    autoFollowRef,
    loadingOlderHistory: options.loadingOlderHistory,
    rowCount: options.rowCount,
    scrollElementRef,
    skipOlderHistoryLoadOnceRef,
  });
  useMountedFlag(mountedRef);

  const postSendFocus = useMessageListPostSendFocus({
    autoFollowRef,
    layoutSignature: options.layoutSignature,
    loadingOlderHistory: options.loadingOlderHistory,
    postSendFocusRequest: options.postSendFocusRequest ?? null,
    scheduleBottomFollow,
    scrollElementRef,
    setShowScrollToBottom,
    syncCurrentBottomAffordance,
    visibleCommittedMessages: options.visibleCommittedMessages,
  });

  const scrollToBottom = useCallback(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    cancelScrollFrame(scrollFrameRef);
    postSendFocus.cancelAnchorScroll();
    postSendFocus.releasePostSendLock(true);
    autoFollowRef.current = true;
    setShowScrollToBottom(false);
    container.scrollTo({
      behavior: 'smooth',
      top: bottomScrollTop(container),
    });
  }, [postSendFocus]);

  useAutoFollowTracking({
    autoFollowRef,
    postSendLockJustStartedRef: postSendFocus.postSendLockJustStartedRef,
    postSendLockRef: postSendFocus.postSendLockRef,
    releasePostSendLock: postSendFocus.releasePostSendLock,
    scrollElementRef,
    setShowScrollToBottom,
  });
  useBottomFollowOnLayoutChange({
    autoFollowRef,
    layoutSignature: options.layoutSignature,
    loadingOlderHistory: options.loadingOlderHistory,
    postSendLockRef: postSendFocus.postSendLockRef,
    scheduleBottomFollow,
  });
  useOlderHistoryLoader({
    ...options,
    mountedRef,
    olderHistoryAnchorRef,
    olderLoadPendingRef,
    scrollElementRef,
    skipOlderHistoryLoadOnceRef,
  });
  useOlderHistoryAnchorCompensation({
    layoutSignature: options.layoutSignature,
    olderHistoryAnchorRef,
    scrollElementRef,
  });
  useScrollFrameCleanup(scrollFrameRef);

  return {
    registerMessageRow: postSendFocus.registerMessageRow,
    historySentinelRef,
    olderHistoryLoadingPaused,
    scrollElementRef,
    scrollToBottom,
    showScrollToBottom,
    trailingSpacerPx: postSendFocus.trailingSpacerPx,
  };
}

function useInitialBottomPlacement(options: {
  autoFollowRef: MutableRefObject<boolean>;
  loadingOlderHistory: boolean;
  rowCount: number;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  skipOlderHistoryLoadOnceRef: MutableRefObject<boolean>;
}) {
  const completedRef = useRef(false);

  useLayoutEffect(() => {
    if (completedRef.current || options.loadingOlderHistory || options.rowCount === 0) {
      return;
    }

    const container = options.scrollElementRef.current;
    if (!container) {
      return;
    }

    container.scrollTop = bottomScrollTop(container);
    options.autoFollowRef.current = true;
    options.skipOlderHistoryLoadOnceRef.current = true;
    completedRef.current = true;
  }, [
    options.autoFollowRef,
    options.loadingOlderHistory,
    options.rowCount,
    options.scrollElementRef,
    options.skipOlderHistoryLoadOnceRef,
  ]);
}

function useAutoFollowTracking(options: {
  autoFollowRef: MutableRefObject<boolean>;
  postSendLockJustStartedRef: MutableRefObject<boolean>;
  postSendLockRef: MutableRefObject<unknown | null>;
  releasePostSendLock: (restoreBottom: boolean) => void;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  setShowScrollToBottom: (value: boolean) => void;
}) {
  const {
    autoFollowRef,
    postSendLockJustStartedRef,
    postSendLockRef,
    releasePostSendLock,
    scrollElementRef,
    setShowScrollToBottom,
  } = options;
  const userScrollIntentRef = useRef(false);
  const syncAutoFollow = useCallback(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }
    if (postSendLockRef.current) {
      if (postSendLockJustStartedRef.current) {
        setShowScrollToBottom(false);
        return;
      }

      const atBottom = isAtHardBottom(container);
      if (atBottom && userScrollIntentRef.current) {
        userScrollIntentRef.current = false;
        releasePostSendLock(true);
        autoFollowRef.current = true;
        setShowScrollToBottom(false);
        return;
      }
      if (userScrollIntentRef.current) {
        userScrollIntentRef.current = false;
        releasePostSendLock(false);
        autoFollowRef.current = false;
        setShowScrollToBottom(true);
        return;
      }
      setShowScrollToBottom(false);
      return;
    }

    userScrollIntentRef.current = false;
    syncBottomAffordance(container, autoFollowRef, setShowScrollToBottom);
  }, [
    autoFollowRef,
    postSendLockJustStartedRef,
    postSendLockRef,
    releasePostSendLock,
    scrollElementRef,
    setShowScrollToBottom,
  ]);
  const markUserScrollIntent = useCallback(() => {
    userScrollIntentRef.current = true;
  }, []);

  useEffect(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    syncAutoFollow();
    container.addEventListener('scroll', syncAutoFollow, { passive: true });
    container.addEventListener('wheel', markUserScrollIntent, { passive: true });
    container.addEventListener('touchmove', markUserScrollIntent, { passive: true });
    return () => {
      container.removeEventListener('scroll', syncAutoFollow);
      container.removeEventListener('wheel', markUserScrollIntent);
      container.removeEventListener('touchmove', markUserScrollIntent);
    };
  }, [markUserScrollIntent, scrollElementRef, syncAutoFollow]);
}

function useBottomFollowOnLayoutChange(options: {
  autoFollowRef: MutableRefObject<boolean>;
  layoutSignature: string;
  loadingOlderHistory: boolean;
  postSendLockRef: MutableRefObject<unknown | null>;
  scheduleBottomFollow: () => void;
}) {
  const {
    autoFollowRef,
    layoutSignature,
    loadingOlderHistory,
    postSendLockRef,
    scheduleBottomFollow,
  } = options;

  useLayoutEffect(() => {
    if (loadingOlderHistory || postSendLockRef.current || !autoFollowRef.current) {
      return;
    }

    scheduleBottomFollow();
  }, [
    autoFollowRef,
    layoutSignature,
    loadingOlderHistory,
    postSendLockRef,
    scheduleBottomFollow,
  ]);
}

function useOlderHistoryLoader(options: UseOlderHistoryLoaderOptions) {
  const {
    hasOlderHistory,
    loadingOlderHistory,
    mountedRef,
    olderHistoryAnchorRef,
    olderLoadPendingRef,
    loadOlderHistory,
    scrollElementRef,
    skipOlderHistoryLoadOnceRef,
  } = options;

  const handleLoadOlderHistory = useCallback(async () => {
    if (skipOlderHistoryLoadOnceRef.current) {
      skipOlderHistoryLoadOnceRef.current = false;
      return;
    }
    if (loadingOlderHistory || olderLoadPendingRef.current || !hasOlderHistory) {
      return;
    }

    const container = scrollElementRef.current;
    if (!container || container.scrollTop > OLDER_HISTORY_TOP_THRESHOLD_PX) {
      return;
    }

    olderLoadPendingRef.current = true;
    olderHistoryAnchorRef.current = {
      scrollHeight: container.scrollHeight,
      scrollTop: container.scrollTop,
    };
    try {
      await loadOlderHistory();
    } finally {
      olderLoadPendingRef.current = false;
      if (!mountedRef.current) {
        olderHistoryAnchorRef.current = null;
      }
    }
  }, [
    hasOlderHistory,
    loadOlderHistory,
    loadingOlderHistory,
    mountedRef,
    olderHistoryAnchorRef,
    olderLoadPendingRef,
    scrollElementRef,
    skipOlderHistoryLoadOnceRef,
  ]);

  useEffect(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    const handleScroll = () => {
      void handleLoadOlderHistory();
    };

    handleScroll();
    container.addEventListener('scroll', handleScroll, { passive: true });
    return () => {
      container.removeEventListener('scroll', handleScroll);
    };
  }, [handleLoadOlderHistory, scrollElementRef]);
}

function useOlderHistoryAnchorCompensation(options: {
  layoutSignature: string;
  olderHistoryAnchorRef: MutableRefObject<OlderHistoryAnchorSnapshot | null>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
}) {
  const { layoutSignature, olderHistoryAnchorRef, scrollElementRef } = options;

  useLayoutEffect(() => {
    const snapshot = olderHistoryAnchorRef.current;
    const container = scrollElementRef.current;
    if (!snapshot || !container) {
      return;
    }

    const heightDelta = container.scrollHeight - snapshot.scrollHeight;
    container.scrollTop = snapshot.scrollTop + heightDelta;
    olderHistoryAnchorRef.current = null;
  }, [layoutSignature, olderHistoryAnchorRef, scrollElementRef]);
}

function useScrollFrameCleanup(scrollFrameRef: MutableRefObject<ScrollFrameHandle | null>) {
  useEffect(() => {
    return () => {
      cancelScrollFrame(scrollFrameRef);
    };
  }, [scrollFrameRef]);
}

function useMountedFlag(mountedRef: MutableRefObject<boolean>) {
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, [mountedRef]);
}

interface UseOlderHistoryLoaderOptions extends UseMessageListScrollOptions {
  mountedRef: MutableRefObject<boolean>;
  olderHistoryAnchorRef: MutableRefObject<OlderHistoryAnchorSnapshot | null>;
  olderLoadPendingRef: MutableRefObject<boolean>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  skipOlderHistoryLoadOnceRef: MutableRefObject<boolean>;
}

function syncBottomAffordance(
  container: HTMLElement,
  autoFollowRef: MutableRefObject<boolean>,
  setShowScrollToBottom: (value: boolean) => void,
) {
  const nextAutoFollow = resolveMessageListAutoFollow(container);
  autoFollowRef.current = nextAutoFollow;
  setShowScrollToBottom(!nextAutoFollow);
}

function scheduleScrollToBottom(options: {
  autoFollowRef: MutableRefObject<boolean>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  scrollFrameRef: MutableRefObject<ScrollFrameHandle | null>;
}) {
  const container = options.scrollElementRef.current;
  if (!container || !options.autoFollowRef.current) {
    return;
  }

  cancelScrollFrame(options.scrollFrameRef);
  options.scrollFrameRef.current = requestScrollFrame(() => {
    options.scrollFrameRef.current = null;
    scrollToBottomIfStillFollowing(options);
  });
}

function scrollToBottomIfStillFollowing(options: {
  autoFollowRef: MutableRefObject<boolean>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
}) {
  const container = options.scrollElementRef.current;
  if (!container || !options.autoFollowRef.current) {
    return;
  }

  container.scrollTop = bottomScrollTop(container);
}

function bottomScrollTop(container: HTMLElement): number {
  return Math.max(0, container.scrollHeight - container.clientHeight);
}

function isAtHardBottom(container: HTMLElement): boolean {
  return Math.abs(bottomScrollTop(container) - container.scrollTop) <= HARD_BOTTOM_TOLERANCE_PX;
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
