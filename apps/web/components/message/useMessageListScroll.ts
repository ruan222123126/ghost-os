import type { MutableRefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { StreamingMessageRow } from '@/lib/chat-view/types';
import type { ChatMessage } from '@/lib/types';
import type { PostSendFocusRequest } from '@/hooks/chat/types';
import { resolveMessageListAutoFollow } from './messageListScroll';
import { useMessageListPostSendFocus } from './useMessageListPostSendFocus';

const OLDER_HISTORY_LOADING_PAUSE_MS = 1500;
type ScrollFrameHandle = number | ReturnType<typeof setTimeout>;

interface OlderLoadPauseHandle {
  resolve: ((completed: boolean) => void) | null;
  timer: ReturnType<typeof setTimeout> | null;
}

export interface UseMessageListScrollOptions {
  hasOlderHistory: boolean;
  loadingOlderHistory: boolean;
  loadOlderHistory: () => Promise<void>;
  layoutSignature: string;
  postSendFocusRequest?: PostSendFocusRequest | null;
  rowCount: number;
  streamingRows: StreamingMessageRow[];
  visibleCommittedMessages: ChatMessage[];
}

export function useMessageListScroll(options: UseMessageListScrollOptions) {
  const historySentinelRef = useRef<HTMLDivElement>(null);
  const scrollElementRef = useRef<HTMLDivElement>(null);
  const [olderHistoryLoadingPaused, setOlderHistoryLoadingPaused] = useState(false);
  const [showScrollToBottom, setShowScrollToBottom] = useState(false);
  const autoFollowRef = useRef(true);
  const mountedRef = useRef(false);
  const olderLoadPendingRef = useRef(false);
  const olderLoadPauseRef = useRef<OlderLoadPauseHandle>({ resolve: null, timer: null });
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
    loadingOlderHistory: options.loadingOlderHistory || olderHistoryLoadingPaused,
    rowCount: options.rowCount,
    scrollElementRef,
    skipOlderHistoryLoadOnceRef,
  });
  useMountedFlag(mountedRef, olderLoadPauseRef);

  const postSendFocus = useMessageListPostSendFocus({
    autoFollowRef,
    layoutSignature: options.layoutSignature,
    loadingOlderHistory: options.loadingOlderHistory || olderHistoryLoadingPaused,
    postSendFocusRequest: options.postSendFocusRequest ?? null,
    scheduleBottomFollow,
    scrollElementRef,
    setShowScrollToBottom,
    streamingRows: options.streamingRows,
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
    postSendFocus.releasePostSendLock(false);
    autoFollowRef.current = true;
    setShowScrollToBottom(false);
    container.scrollTo({
      behavior: 'smooth',
      top: 0,
    });
  }, [postSendFocus]);

  useAutoFollowTracking({
    autoFollowRef,
    postSendLockJustStartedRef: postSendFocus.postSendLockJustStartedRef,
    postSendLockRef: postSendFocus.postSendLockRef,
    scrollElementRef,
    setShowScrollToBottom,
    syncPostSendLock: postSendFocus.syncPostSendLock,
  });
  useBottomFollowOnLayoutChange({
    autoFollowRef,
    layoutSignature: options.layoutSignature,
    loadingOlderHistory: options.loadingOlderHistory || olderHistoryLoadingPaused,
    scheduleBottomFollow,
  });
  useOlderHistoryObserver({
    ...options,
    historySentinelRef,
    mountedRef,
    olderHistoryLoadingPaused,
    olderLoadPendingRef,
    olderLoadPauseRef,
    scrollElementRef,
    setOlderHistoryLoadingPaused,
    skipOlderHistoryLoadOnceRef,
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

    container.scrollTop = 0;
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
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  setShowScrollToBottom: (value: boolean) => void;
  syncPostSendLock: () => void;
}) {
  const {
    autoFollowRef,
    postSendLockJustStartedRef,
    postSendLockRef,
    scrollElementRef,
    setShowScrollToBottom,
    syncPostSendLock,
  } = options;
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

      syncPostSendLock();
      setShowScrollToBottom(false);
      return;
    }

    syncBottomAffordance(container, autoFollowRef, setShowScrollToBottom);
  }, [
    autoFollowRef,
    postSendLockJustStartedRef,
    postSendLockRef,
    scrollElementRef,
    setShowScrollToBottom,
    syncPostSendLock,
  ]);

  useEffect(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    syncAutoFollow();
    container.addEventListener('scroll', syncAutoFollow, { passive: true });
    return () => {
      container.removeEventListener('scroll', syncAutoFollow);
    };
  }, [scrollElementRef, syncAutoFollow]);
}

function useBottomFollowOnLayoutChange(options: {
  autoFollowRef: MutableRefObject<boolean>;
  layoutSignature: string;
  loadingOlderHistory: boolean;
  scheduleBottomFollow: () => void;
}) {
  const {
    autoFollowRef,
    layoutSignature,
    loadingOlderHistory,
    scheduleBottomFollow,
  } = options;

  useLayoutEffect(() => {
    if (loadingOlderHistory || !autoFollowRef.current) {
      return;
    }

    scheduleBottomFollow();
  }, [
    autoFollowRef,
    layoutSignature,
    loadingOlderHistory,
    scheduleBottomFollow,
  ]);
}

function useOlderHistoryObserver(options: UseOlderHistoryObserverOptions) {
  const {
    hasOlderHistory,
    historySentinelRef,
    loadingOlderHistory,
    mountedRef,
    olderHistoryLoadingPaused,
    olderLoadPendingRef,
    olderLoadPauseRef,
    loadOlderHistory,
    scrollElementRef,
    setOlderHistoryLoadingPaused,
    skipOlderHistoryLoadOnceRef,
  } = options;
  const handleLoadOlderHistory = useCallback(async () => {
    if (skipOlderHistoryLoadOnceRef.current) {
      skipOlderHistoryLoadOnceRef.current = false;
      return;
    }
    if (loadingOlderHistory || olderHistoryLoadingPaused || olderLoadPendingRef.current) {
      return;
    }
    if (!hasOlderHistory) {
      return;
    }

    olderLoadPendingRef.current = true;
    setOlderHistoryLoadingPaused(true);
    try {
      const completedPause = await pauseOlderHistoryLoading(olderLoadPauseRef);
      if (!completedPause) {
        return;
      }

      await loadOlderHistory();
    } finally {
      olderLoadPendingRef.current = false;
      if (mountedRef.current) {
        setOlderHistoryLoadingPaused(false);
      }
    }
  }, [
    hasOlderHistory,
    loadOlderHistory,
    loadingOlderHistory,
    mountedRef,
    olderHistoryLoadingPaused,
    olderLoadPendingRef,
    olderLoadPauseRef,
    setOlderHistoryLoadingPaused,
    skipOlderHistoryLoadOnceRef,
  ]);

  useEffect(() => {
    const root = scrollElementRef.current;
    const target = historySentinelRef.current;
    if (!root || !target) {
      return;
    }

    const observer = new IntersectionObserver((entries) => {
      if (!entries[0]?.isIntersecting) {
        return;
      }

      void handleLoadOlderHistory();
    }, {
      root,
      threshold: 0.1,
    });

    observer.observe(target);
    return () => {
      observer.disconnect();
    };
  }, [handleLoadOlderHistory, historySentinelRef, scrollElementRef]);
}

function useScrollFrameCleanup(scrollFrameRef: MutableRefObject<ScrollFrameHandle | null>) {
  useEffect(() => {
    return () => {
      cancelScrollFrame(scrollFrameRef);
    };
  }, [scrollFrameRef]);
}

function useMountedFlag(
  mountedRef: MutableRefObject<boolean>,
  pauseRef: MutableRefObject<OlderLoadPauseHandle>,
) {
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      cancelOlderHistoryLoadingPause(pauseRef);
    };
  }, [mountedRef, pauseRef]);
}

interface UseOlderHistoryObserverOptions extends UseMessageListScrollOptions {
  historySentinelRef: MutableRefObject<HTMLDivElement | null>;
  mountedRef: MutableRefObject<boolean>;
  olderHistoryLoadingPaused: boolean;
  olderLoadPendingRef: MutableRefObject<boolean>;
  olderLoadPauseRef: MutableRefObject<OlderLoadPauseHandle>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  setOlderHistoryLoadingPaused: (value: boolean) => void;
  skipOlderHistoryLoadOnceRef: MutableRefObject<boolean>;
}

function pauseOlderHistoryLoading(
  pauseRef: MutableRefObject<OlderLoadPauseHandle>,
) {
  return new Promise<boolean>((resolve) => {
    pauseRef.current.resolve = resolve;
    pauseRef.current.timer = setTimeout(() => {
      pauseRef.current.resolve = null;
      pauseRef.current.timer = null;
      resolve(true);
    }, OLDER_HISTORY_LOADING_PAUSE_MS);
  });
}

function cancelOlderHistoryLoadingPause(
  pauseRef: MutableRefObject<OlderLoadPauseHandle>,
) {
  if (pauseRef.current.timer !== null) {
    clearTimeout(pauseRef.current.timer);
    pauseRef.current.timer = null;
  }

  const resolve = pauseRef.current.resolve;
  pauseRef.current.resolve = null;
  resolve?.(false);
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

  container.scrollTop = 0;
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
