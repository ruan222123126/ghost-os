import type { Virtualizer } from '@tanstack/react-virtual';
import type { MutableRefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef } from 'react';
import {
  getPostSendLockedScrollTop,
  type PostSendFollowTrackingState,
  resolveMessageListAutoFollow,
  shouldClearPostSendProgrammaticScrollTarget,
  shouldAdjustScrollPositionOnItemSizeChange,
} from './messageListScroll';
import { useMessageListPostSendFocus } from './useMessageListPostSendFocus';

const LOAD_OLDER_TRIGGER_ROWS = 5;
const MANUAL_SCROLL_INTENT_CLEAR_DELAY_MS = 500;
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

interface PrependAnchor {
  firstVisibleCommittedMessageId: string | null;
  loadingOffsetPx: number;
  scrollHeight: number;
  scrollTop: number;
  visibleCommittedMessageCount: number;
}

export interface UseMessageListScrollOptions {
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
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
  const autoFollowRef = useRef(true);
  const olderLoadPendingRef = useRef(false);
  const prependAnchorRef = useRef<PrependAnchor | null>(null);
  const skipOlderHistoryLoadOnceRef = useRef(false);
  const scrollFrameRef = useRef<number | null>(null);
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

  useVirtualizerSizeAdjustment(options.rowVirtualizer, autoFollowRef, postSendFollowTrackingRef);
  useAutoFollowTracking(
    scrollElementRef,
    autoFollowRef,
    postSendFollowTrackingRef,
    setPostSendFollowTracking,
  );
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
    olderLoadPendingRef,
    prependAnchorRef,
    scrollElementRef,
    skipOlderHistoryLoadOnceRef,
  });
  usePrependAnchorRestore({ ...options, prependAnchorRef, scrollElementRef });
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

  return { scrollElementRef, trailingSpacerPx };
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

function useVirtualizerSizeAdjustment(
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>,
  autoFollowRef: MutableRefObject<boolean>,
  postSendFollowTrackingRef: MutableRefObject<PostSendFollowTrackingState>,
) {
  useLayoutEffect(() => {
    rowVirtualizer.shouldAdjustScrollPositionOnItemSizeChange = () =>
      shouldAdjustScrollPositionOnItemSizeChange(
        autoFollowRef.current,
        postSendFollowTrackingRef.current.mode,
      );

    return () => {
      rowVirtualizer.shouldAdjustScrollPositionOnItemSizeChange = undefined;
    };
  }, [autoFollowRef, postSendFollowTrackingRef, rowVirtualizer]);
}

function useAutoFollowTracking(
  scrollElementRef: MutableRefObject<HTMLDivElement | null>,
  autoFollowRef: MutableRefObject<boolean>,
  postSendFollowTrackingRef: MutableRefObject<PostSendFollowTrackingState>,
  setPostSendFollowTracking: (value: PostSendFollowTrackingState) => void,
) {
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
  }, [autoFollowRef, postSendFollowTrackingRef, scrollElementRef, setPostSendFollowTracking]);

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
    firstVirtualItemIndex,
    firstVisibleCommittedMessageId,
    hasOlderHistory,
    loadOlderHistory,
    loadingOlderHistory,
    olderLoadPendingRef,
    prependAnchorRef,
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

    capturePrependAnchor({
      container: scrollElementRef.current,
      firstVisibleCommittedMessageId,
      prependAnchorRef,
      visibleCommittedMessageCount,
    });
    olderLoadPendingRef.current = true;
    try {
      await loadOlderHistory();
    } finally {
      olderLoadPendingRef.current = false;
    }
  }, [
    firstVisibleCommittedMessageId,
    hasOlderHistory,
    loadOlderHistory,
    loadingOlderHistory,
    olderLoadPendingRef,
    prependAnchorRef,
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
      container: options.scrollElementRef.current,
      firstVisibleCommittedMessageId: options.firstVisibleCommittedMessageId,
      loadingOlderHistory: options.loadingOlderHistory,
      prependAnchorRef: options.prependAnchorRef,
      visibleCommittedMessageCount: options.visibleCommittedMessageCount,
    });
  }, [
    options.firstVisibleCommittedMessageId,
    options.loadingOlderHistory,
    options.prependAnchorRef,
    options.scrollElementRef,
    options.visibleCommittedMessageCount,
  ]);
}

function useScrollFrameCleanup(scrollFrameRef: MutableRefObject<number | null>) {
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
  olderLoadPendingRef: MutableRefObject<boolean>;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  skipOlderHistoryLoadOnceRef: MutableRefObject<boolean>;
}

interface PrependAnchorRestoreOptions extends UseMessageListScrollOptions {
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
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

function capturePrependAnchor(options: {
  container: HTMLDivElement | null;
  firstVisibleCommittedMessageId: string | null;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  visibleCommittedMessageCount: number;
}) {
  const {
    container,
    firstVisibleCommittedMessageId,
    prependAnchorRef,
    visibleCommittedMessageCount,
  } = options;
  if (!container) {
    return;
  }

  prependAnchorRef.current = {
    firstVisibleCommittedMessageId,
    loadingOffsetPx: 0,
    scrollHeight: container.scrollHeight,
    scrollTop: container.scrollTop,
    visibleCommittedMessageCount,
  };
}

function restorePrependAnchor(options: {
  container: HTMLDivElement | null;
  firstVisibleCommittedMessageId: string | null;
  loadingOlderHistory: boolean;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  visibleCommittedMessageCount: number;
}) {
  const {
    container,
    firstVisibleCommittedMessageId,
    loadingOlderHistory,
    prependAnchorRef,
    visibleCommittedMessageCount,
  } = options;
  const anchor = prependAnchorRef.current;
  if (!anchor || !container) {
    return;
  }
  if (loadingOlderHistory) {
    applyPrependAnchorOffset(container, anchor);
    return;
  }
  const hasPrependedVisibleMessages = visibleCommittedMessageCount > anchor.visibleCommittedMessageCount
    && firstVisibleCommittedMessageId !== anchor.firstVisibleCommittedMessageId;
  if (!hasPrependedVisibleMessages) {
    restorePrependAnchorAfterSkippedLoad(container, anchor);
    prependAnchorRef.current = null;
    return;
  }

  applyPrependAnchorOffset(container, anchor);
  prependAnchorRef.current = null;
}

function applyPrependAnchorOffset(container: HTMLDivElement, anchor: PrependAnchor) {
  const offsetPx = container.scrollHeight - anchor.scrollHeight;
  if (offsetPx === 0) {
    return;
  }

  container.scrollTop = anchor.scrollTop + offsetPx;
  anchor.scrollHeight = container.scrollHeight;
  anchor.scrollTop = container.scrollTop;
  anchor.loadingOffsetPx += offsetPx;
}

function restorePrependAnchorAfterSkippedLoad(container: HTMLDivElement, anchor: PrependAnchor) {
  if (anchor.loadingOffsetPx === 0) {
    return;
  }

  container.scrollTop = anchor.scrollTop - anchor.loadingOffsetPx;
}

function scheduleScrollToBottom(options: {
  autoFollowRef: MutableRefObject<boolean>;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  scrollFrameRef: MutableRefObject<number | null>;
}) {
  const container = options.scrollElementRef.current;
  if (!container || options.prependAnchorRef.current || !options.autoFollowRef.current) {
    return;
  }

  cancelScrollFrame(options.scrollFrameRef);
  options.scrollFrameRef.current = window.requestAnimationFrame(() => {
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

function cancelScrollFrame(scrollFrameRef: MutableRefObject<number | null>) {
  if (scrollFrameRef.current === null) {
    return;
  }

  window.cancelAnimationFrame(scrollFrameRef.current);
  scrollFrameRef.current = null;
}
