import type { Virtualizer } from '@tanstack/react-virtual';
import type { MutableRefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import {
  getPostSendLockedScrollTop,
  type PostSendFollowTrackingState,
  resolveMessageListAutoFollow,
  shouldClearPostSendProgrammaticScrollTarget,
  shouldAdjustScrollPositionOnItemSizeChange,
} from './messageListScroll';
import {
  capturePrependAnchor,
  type PrependAnchor,
  restorePrependAnchorAfterSkippedLoad,
  restorePrependAnchorPosition,
} from './messageListPrependAnchor';
import { useMessageListPostSendFocus } from './useMessageListPostSendFocus';

const LOAD_OLDER_TRIGGER_ROWS = 5;
const MANUAL_SCROLL_INTENT_CLEAR_DELAY_MS = 500;
const PREPEND_ANCHOR_SETTLE_FRAMES = 3;
type ScrollFrameHandle = number | ReturnType<typeof setTimeout>;
const MANUAL_SCROLL_KEYS = new Set([
  'ArrowDown',
  'ArrowUp',
  'End',
  'Home',
  'PageDown',
  'PageUp',
  ' ',
  'Spacebar',
]);

export interface UseMessageListScrollOptions {
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  rowKeys: readonly string[];
  firstVirtualItemIndex: number | null;
  firstVisibleCommittedMessageId: string | null;
  hasOlderHistory: boolean;
  loadingOlderHistory: boolean;
  loadOlderHistory: () => Promise<void>;
  visibleCommittedMessageCount: number;
  layoutSignature: string;
  postSendAnchorIndex?: number | null;
  postSendHasVisibleContent?: boolean;
  postSendToken?: number;
}

export function useMessageListScroll(options: UseMessageListScrollOptions) {
  const scrollElementRef = useRef<HTMLDivElement>(null);
  const [showScrollToBottom, setShowScrollToBottom] = useState(false);
  const autoFollowRef = useRef(true);
  const olderLoadPendingRef = useRef(false);
  const prependAnchorRef = useRef<PrependAnchor | null>(null);
  const prependAnchorReleaseFrameRef = useRef<ScrollFrameHandle | null>(null);
  const skipOlderHistoryLoadOnceRef = useRef(false);
  const scrollFrameRef = useRef<ScrollFrameHandle | null>(null);
  const postSendFollowTrackingRef = useRef<PostSendFollowTrackingState>({
    mode: 'idle',
    controlledScrollTopPx: null,
  });
  const setPostSendFollowTracking = useCallback((value: PostSendFollowTrackingState) => {
    postSendFollowTrackingRef.current = value;
  }, []);
  const scheduleNormalFollow = useCallback(() => {
    scheduleScrollToBottom({
      autoFollowRef,
      prependAnchorRef,
      scrollElementRef,
      scrollFrameRef,
    });
  }, []);
  const cancelScheduledScroll = useCallback(() => {
    cancelScrollFrame(scrollFrameRef);
  }, []);
  const cancelPrependAnchorRelease = useCallback(() => {
    cancelScrollFrame(prependAnchorReleaseFrameRef);
  }, []);
  const schedulePrependAnchorRelease = useCallback(() => {
    cancelPrependAnchorRelease();
    let remainingFrames = PREPEND_ANCHOR_SETTLE_FRAMES;
    const releaseAfterMeasurementFrames = () => {
      remainingFrames -= 1;
      if (remainingFrames <= 0) {
        prependAnchorReleaseFrameRef.current = null;
        prependAnchorRef.current = null;
        return;
      }

      prependAnchorReleaseFrameRef.current = requestScrollFrame(releaseAfterMeasurementFrames);
    };

    prependAnchorReleaseFrameRef.current = requestScrollFrame(releaseAfterMeasurementFrames);
  }, [cancelPrependAnchorRelease]);
  const scrollToBottom = useCallback(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    cancelPrependAnchorRelease();
    cancelScrollFrame(scrollFrameRef);
    prependAnchorRef.current = null;
    autoFollowRef.current = true;
    setShowScrollToBottom(false);
    setPostSendFollowTracking({
      mode: 'idle',
      controlledScrollTopPx: null,
      programmaticScrollTargetPx: null,
    });
    container.scrollTo({
      behavior: 'smooth',
      top: container.scrollHeight,
    });
  }, [cancelPrependAnchorRelease, setPostSendFollowTracking]);

  useVirtualizerSizeAdjustment({
    autoFollowRef,
    postSendFollowTrackingRef,
    prependAnchorRef,
    rowVirtualizer: options.rowVirtualizer,
    scrollElementRef,
  });
  useAutoFollowTracking({
    autoFollowRef,
    postSendFollowTrackingRef,
    scrollElementRef,
    setPostSendFollowTracking,
    setShowScrollToBottom,
  });
  useInitialBottomScroll({
    autoFollowRef,
    loadingOlderHistory: options.loadingOlderHistory,
    prependAnchorRef,
    scrollElementRef,
    skipOlderHistoryLoadOnceRef,
    visibleCommittedMessageCount: options.visibleCommittedMessageCount,
  });
  useOlderHistoryLoading({
    ...options,
    cancelPrependAnchorRelease,
    olderLoadPendingRef,
    prependAnchorRef,
    scrollElementRef,
    skipOlderHistoryLoadOnceRef,
  });
  usePrependAnchorRestore({
    ...options,
    cancelPrependAnchorRelease,
    prependAnchorRef,
    schedulePrependAnchorRelease,
    scrollElementRef,
  });
  const { trailingSpacerPx } = useMessageListPostSendFocus({
    autoFollowRef,
    cancelScheduledScroll,
    layoutSignature: options.layoutSignature,
    loadingOlderHistory: options.loadingOlderHistory,
    onNormalLayoutChange: scheduleNormalFollow,
    postSendAnchorIndex: options.postSendAnchorIndex,
    postSendHasVisibleContent: options.postSendHasVisibleContent,
    postSendToken: options.postSendToken,
    rowVirtualizer: options.rowVirtualizer,
    setPostSendFollowTracking,
    scrollElementRef,
  });
  usePostSendScrollLock(scrollElementRef, postSendFollowTrackingRef);
  useScrollFrameCleanup(scrollFrameRef);
  useScrollFrameCleanup(prependAnchorReleaseFrameRef);

  return { scrollElementRef, scrollToBottom, showScrollToBottom, trailingSpacerPx };
}

function useInitialBottomScroll(options: {
  autoFollowRef: MutableRefObject<boolean>;
  loadingOlderHistory: boolean;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  skipOlderHistoryLoadOnceRef: MutableRefObject<boolean>;
  visibleCommittedMessageCount: number;
}) {
  const completedRef = useRef(false);

  useLayoutEffect(() => {
    if (completedRef.current || options.loadingOlderHistory) {
      return;
    }
    if (options.visibleCommittedMessageCount === 0 || options.prependAnchorRef.current) {
      return;
    }

    const container = options.scrollElementRef.current;
    if (!container) {
      return;
    }

    container.scrollTop = container.scrollHeight;
    options.autoFollowRef.current = true;
    options.skipOlderHistoryLoadOnceRef.current = true;
    completedRef.current = true;
  }, [
    options.autoFollowRef,
    options.loadingOlderHistory,
    options.prependAnchorRef,
    options.scrollElementRef,
    options.skipOlderHistoryLoadOnceRef,
    options.visibleCommittedMessageCount,
  ]);
}

function useVirtualizerSizeAdjustment(options: {
  autoFollowRef: MutableRefObject<boolean>;
  postSendFollowTrackingRef: MutableRefObject<PostSendFollowTrackingState>;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
}) {
  useLayoutEffect(() => {
    options.rowVirtualizer.shouldAdjustScrollPositionOnItemSizeChange = (item) =>
      shouldAdjustScrollPositionOnItemSizeChange(
        options.autoFollowRef.current,
        options.postSendFollowTrackingRef.current.mode,
        {
          active: options.prependAnchorRef.current !== null,
          itemStartPx: item.start,
          scrollTopPx: options.scrollElementRef.current?.scrollTop ?? 0,
        },
      );

    return () => {
      options.rowVirtualizer.shouldAdjustScrollPositionOnItemSizeChange = undefined;
    };
  }, [
    options.autoFollowRef,
    options.postSendFollowTrackingRef,
    options.prependAnchorRef,
    options.rowVirtualizer,
    options.scrollElementRef,
  ]);
}

function useAutoFollowTracking(options: {
  autoFollowRef: MutableRefObject<boolean>;
  postSendFollowTrackingRef: MutableRefObject<PostSendFollowTrackingState>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  setPostSendFollowTracking: (value: PostSendFollowTrackingState) => void;
  setShowScrollToBottom: (value: boolean) => void;
}) {
  const {
    autoFollowRef,
    postSendFollowTrackingRef,
    scrollElementRef,
    setPostSendFollowTracking,
    setShowScrollToBottom,
  } = options;
  const manualScrollIntentRef = useRef(false);
  const manualScrollIntentTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const syncAutoFollow = useCallback(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    const tracking = postSendFollowTrackingRef.current;
    const nextAutoFollow = resolveMessageListAutoFollow(container, tracking, {
      manualScrollIntent: manualScrollIntentRef.current,
    });
    autoFollowRef.current = nextAutoFollow;
    setShowScrollToBottom(!nextAutoFollow);
    if (shouldClearPostSendProgrammaticScrollTarget(container, tracking)) {
      setPostSendFollowTracking({
        ...tracking,
        programmaticScrollTargetPx: null,
      });
      return;
    }
    if (!nextAutoFollow && tracking.mode !== 'idle') {
      setPostSendFollowTracking({
        mode: 'idle',
        controlledScrollTopPx: null,
        programmaticScrollTargetPx: null,
      });
    }
  }, [
    autoFollowRef,
    postSendFollowTrackingRef,
    scrollElementRef,
    setPostSendFollowTracking,
    setShowScrollToBottom,
  ]);

  const clearProgrammaticScrollTarget = useCallback(() => {
    const tracking = postSendFollowTrackingRef.current;
    if (
      tracking.mode === 'idle'
      || tracking.programmaticScrollTargetPx === null
      || tracking.programmaticScrollTargetPx === undefined
    ) {
      return;
    }

    setPostSendFollowTracking({
      ...tracking,
      programmaticScrollTargetPx: null,
    });
  }, [postSendFollowTrackingRef, setPostSendFollowTracking]);

  const clearManualScrollIntentTimer = useCallback(() => {
    if (manualScrollIntentTimerRef.current === null) {
      return;
    }

    clearTimeout(manualScrollIntentTimerRef.current);
    manualScrollIntentTimerRef.current = null;
  }, []);

  const markManualScrollIntent = useCallback(() => {
    manualScrollIntentRef.current = true;
    clearProgrammaticScrollTarget();
    clearManualScrollIntentTimer();
    manualScrollIntentTimerRef.current = setTimeout(() => {
      manualScrollIntentRef.current = false;
      manualScrollIntentTimerRef.current = null;
    }, MANUAL_SCROLL_INTENT_CLEAR_DELAY_MS);
  }, [clearManualScrollIntentTimer, clearProgrammaticScrollTarget]);

  const markKeyboardScrollIntent = useCallback((event: KeyboardEvent) => {
    if (!MANUAL_SCROLL_KEYS.has(event.key)) {
      return;
    }

    markManualScrollIntent();
  }, [markManualScrollIntent]);

  useEffect(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    syncAutoFollow();
    container.addEventListener('scroll', syncAutoFollow, { passive: true });
    container.addEventListener('keydown', markKeyboardScrollIntent);
    container.addEventListener('touchmove', markManualScrollIntent, { passive: true });
    container.addEventListener('touchstart', markManualScrollIntent, { passive: true });
    container.addEventListener('wheel', markManualScrollIntent, { passive: true });
    return () => {
      container.removeEventListener('scroll', syncAutoFollow);
      container.removeEventListener('keydown', markKeyboardScrollIntent);
      container.removeEventListener('touchmove', markManualScrollIntent);
      container.removeEventListener('touchstart', markManualScrollIntent);
      container.removeEventListener('wheel', markManualScrollIntent);
      clearManualScrollIntentTimer();
    };
  }, [
    clearManualScrollIntentTimer,
    markKeyboardScrollIntent,
    markManualScrollIntent,
    scrollElementRef,
    syncAutoFollow,
  ]);
}

function useOlderHistoryLoading(options: UseOlderHistoryLoadingOptions) {
  const {
    cancelPrependAnchorRelease,
    firstVirtualItemIndex,
    firstVisibleCommittedMessageId,
    hasOlderHistory,
    loadOlderHistory,
    loadingOlderHistory,
    olderLoadPendingRef,
    prependAnchorRef,
    rowKeys,
    rowVirtualizer,
    scrollElementRef,
    skipOlderHistoryLoadOnceRef,
    visibleCommittedMessageCount,
  } = options;
  const handleLoadOlderHistory = useCallback(async () => {
    if (loadingOlderHistory || olderLoadPendingRef.current) {
      return;
    }
    if (!hasOlderHistory) {
      return;
    }

    cancelPrependAnchorRelease();
    prependAnchorRef.current = capturePrependAnchor({
      container: scrollElementRef.current,
      firstVisibleCommittedMessageId,
      rowKeys,
      rowVirtualizer,
      visibleCommittedMessageCount,
    });
    olderLoadPendingRef.current = true;
    try {
      await loadOlderHistory();
    } finally {
      olderLoadPendingRef.current = false;
    }
  }, [
    cancelPrependAnchorRelease,
    firstVisibleCommittedMessageId,
    hasOlderHistory,
    loadOlderHistory,
    loadingOlderHistory,
    olderLoadPendingRef,
    prependAnchorRef,
    rowKeys,
    rowVirtualizer,
    scrollElementRef,
    visibleCommittedMessageCount,
  ]);

  useEffect(() => {
    if (skipOlderHistoryLoadOnceRef.current) {
      skipOlderHistoryLoadOnceRef.current = false;
      return;
    }
    if (!shouldLoadOlderHistory({ firstVirtualItemIndex, hasOlderHistory, loadingOlderHistory })) {
      return;
    }

    void handleLoadOlderHistory();
  }, [
    firstVirtualItemIndex,
    handleLoadOlderHistory,
    hasOlderHistory,
    loadingOlderHistory,
    skipOlderHistoryLoadOnceRef,
  ]);
}

function usePrependAnchorRestore(options: PrependAnchorRestoreOptions) {
  useLayoutEffect(() => {
    restorePrependAnchor({
      cancelPrependAnchorRelease: options.cancelPrependAnchorRelease,
      container: options.scrollElementRef.current,
      firstVisibleCommittedMessageId: options.firstVisibleCommittedMessageId,
      loadingOlderHistory: options.loadingOlderHistory,
      prependAnchorRef: options.prependAnchorRef,
      rowKeys: options.rowKeys,
      rowVirtualizer: options.rowVirtualizer,
      schedulePrependAnchorRelease: options.schedulePrependAnchorRelease,
      visibleCommittedMessageCount: options.visibleCommittedMessageCount,
    });
  }, [
    options.cancelPrependAnchorRelease,
    options.firstVisibleCommittedMessageId,
    options.loadingOlderHistory,
    options.prependAnchorRef,
    options.rowKeys,
    options.rowVirtualizer,
    options.schedulePrependAnchorRelease,
    options.scrollElementRef,
    options.visibleCommittedMessageCount,
  ]);
}

function useScrollFrameCleanup(scrollFrameRef: MutableRefObject<ScrollFrameHandle | null>) {
  useEffect(() => {
    return () => {
      cancelScrollFrame(scrollFrameRef);
    };
  }, [scrollFrameRef]);
}

function usePostSendScrollLock(
  scrollElementRef: MutableRefObject<HTMLDivElement | null>,
  postSendFollowTrackingRef: MutableRefObject<PostSendFollowTrackingState>,
) {
  useEffect(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    const clampScroll = () => {
      const lockedScrollTop = getPostSendLockedScrollTop(container, postSendFollowTrackingRef.current);
      if (lockedScrollTop === null) {
        return;
      }

      container.scrollTop = lockedScrollTop;
    };

    container.addEventListener('scroll', clampScroll, { passive: true });
    return () => {
      container.removeEventListener('scroll', clampScroll);
    };
  }, [postSendFollowTrackingRef, scrollElementRef]);
}

interface UseOlderHistoryLoadingOptions extends UseMessageListScrollOptions {
  cancelPrependAnchorRelease: () => void;
  olderLoadPendingRef: MutableRefObject<boolean>;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  skipOlderHistoryLoadOnceRef: MutableRefObject<boolean>;
}

interface PrependAnchorRestoreOptions extends UseMessageListScrollOptions {
  cancelPrependAnchorRelease: () => void;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  schedulePrependAnchorRelease: () => void;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
}

function shouldLoadOlderHistory(options: {
  firstVirtualItemIndex: number | null;
  hasOlderHistory: boolean;
  loadingOlderHistory: boolean;
}): boolean {
  if (!options.hasOlderHistory || options.loadingOlderHistory) {
    return false;
  }
  return options.firstVirtualItemIndex !== null
    && options.firstVirtualItemIndex <= LOAD_OLDER_TRIGGER_ROWS;
}

function restorePrependAnchor(options: {
  cancelPrependAnchorRelease: () => void;
  container: HTMLDivElement | null;
  firstVisibleCommittedMessageId: string | null;
  loadingOlderHistory: boolean;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  rowKeys: readonly string[];
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  schedulePrependAnchorRelease: () => void;
  visibleCommittedMessageCount: number;
}) {
  const {
    cancelPrependAnchorRelease,
    container,
    firstVisibleCommittedMessageId,
    loadingOlderHistory,
    prependAnchorRef,
    rowKeys,
    rowVirtualizer,
    schedulePrependAnchorRelease,
    visibleCommittedMessageCount,
  } = options;
  const anchor = prependAnchorRef.current;
  if (!anchor || !container) {
    return;
  }
  if (loadingOlderHistory) {
    restorePrependAnchorPosition({ anchor, container, rowKeys, rowVirtualizer });
    return;
  }
  const hasPrependedVisibleMessages = visibleCommittedMessageCount > anchor.visibleCommittedMessageCount
    && firstVisibleCommittedMessageId !== anchor.firstVisibleCommittedMessageId;
  if (!hasPrependedVisibleMessages) {
    restorePrependAnchorAfterSkippedLoad(container, anchor);
    cancelPrependAnchorRelease();
    prependAnchorRef.current = null;
    return;
  }

  restorePrependAnchorPosition({ anchor, container, rowKeys, rowVirtualizer });
  if (!anchor.settling) {
    anchor.settling = true;
    schedulePrependAnchorRelease();
  }
}

function scheduleScrollToBottom(options: {
  autoFollowRef: MutableRefObject<boolean>;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  scrollFrameRef: MutableRefObject<ScrollFrameHandle | null>;
}) {
  const container = options.scrollElementRef.current;
  if (!container || options.prependAnchorRef.current || !options.autoFollowRef.current) {
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
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
}) {
  const container = options.scrollElementRef.current;
  if (!container || options.prependAnchorRef.current || !options.autoFollowRef.current) {
    return;
  }

  container.scrollTop = container.scrollHeight;
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
