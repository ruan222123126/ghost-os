import type { Virtualizer } from '@tanstack/react-virtual';
import type { MutableRefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef } from 'react';
import {
  isMessageListNearBottom,
  shouldAdjustScrollPositionOnItemSizeChange,
} from './messageListScroll';
import { useMessageListPostSendFocus } from './useMessageListPostSendFocus';

const LOAD_OLDER_TRIGGER_ROWS = 5;

interface PrependAnchor {
  scrollHeight: number;
  scrollTop: number;
}

export interface UseMessageListScrollOptions {
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  firstVirtualItemIndex: number | null;
  hasOlderHistory: boolean;
  loadingOlderHistory: boolean;
  loadOlderHistory: () => Promise<void>;
  visibleCommittedMessageCount: number;
  layoutSignature: string;
  postSendAnchorIndex?: number | null;
  postSendHasVisibleAssistantText?: boolean;
  postSendToken?: number;
}

export function useMessageListScroll(options: UseMessageListScrollOptions) {
  const scrollElementRef = useRef<HTMLDivElement>(null);
  const autoFollowRef = useRef(true);
  const olderLoadPendingRef = useRef(false);
  const prependAnchorRef = useRef<PrependAnchor | null>(null);
  const scrollFrameRef = useRef<number | null>(null);
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

  useVirtualizerSizeAdjustment(options.rowVirtualizer, autoFollowRef);
  useAutoFollowTracking(scrollElementRef, autoFollowRef);
  useOlderHistoryLoading({ ...options, olderLoadPendingRef, prependAnchorRef, scrollElementRef });
  usePrependAnchorRestore({ ...options, prependAnchorRef, scrollElementRef });
  const { trailingSpacerPx } = useMessageListPostSendFocus({
    autoFollowRef,
    cancelScheduledScroll,
    layoutSignature: options.layoutSignature,
    loadingOlderHistory: options.loadingOlderHistory,
    onNormalLayoutChange: scheduleNormalFollow,
    postSendAnchorIndex: options.postSendAnchorIndex,
    postSendHasVisibleAssistantText: options.postSendHasVisibleAssistantText,
    postSendToken: options.postSendToken,
    rowVirtualizer: options.rowVirtualizer,
    scrollElementRef,
  });
  useScrollFrameCleanup(scrollFrameRef);

  return { scrollElementRef, trailingSpacerPx };
}

function useVirtualizerSizeAdjustment(
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>,
  autoFollowRef: MutableRefObject<boolean>,
) {
  useLayoutEffect(() => {
    rowVirtualizer.shouldAdjustScrollPositionOnItemSizeChange = () =>
      shouldAdjustScrollPositionOnItemSizeChange(autoFollowRef.current);

    return () => {
      rowVirtualizer.shouldAdjustScrollPositionOnItemSizeChange = undefined;
    };
  }, [autoFollowRef, rowVirtualizer]);
}

function useAutoFollowTracking(
  scrollElementRef: MutableRefObject<HTMLDivElement | null>,
  autoFollowRef: MutableRefObject<boolean>,
) {
  const syncAutoFollow = useCallback(() => {
    const container = scrollElementRef.current;
    if (container) {
      autoFollowRef.current = isMessageListNearBottom(container);
    }
  }, [autoFollowRef, scrollElementRef]);

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

function useOlderHistoryLoading(options: UseOlderHistoryLoadingOptions) {
  const {
    firstVirtualItemIndex,
    hasOlderHistory,
    loadOlderHistory,
    loadingOlderHistory,
    olderLoadPendingRef,
    prependAnchorRef,
    scrollElementRef,
  } = options;
  const handleLoadOlderHistory = useCallback(async () => {
    if (loadingOlderHistory || olderLoadPendingRef.current) {
      return;
    }
    if (!hasOlderHistory) {
      return;
    }

    capturePrependAnchor(scrollElementRef.current, prependAnchorRef);
    olderLoadPendingRef.current = true;
    try {
      await loadOlderHistory();
    } finally {
      olderLoadPendingRef.current = false;
    }
  }, [
    hasOlderHistory,
    loadOlderHistory,
    loadingOlderHistory,
    olderLoadPendingRef,
    prependAnchorRef,
    scrollElementRef,
  ]);

  useEffect(() => {
    if (!shouldLoadOlderHistory({ firstVirtualItemIndex, hasOlderHistory, loadingOlderHistory })) {
      return;
    }

    void handleLoadOlderHistory();
  }, [firstVirtualItemIndex, handleLoadOlderHistory, hasOlderHistory, loadingOlderHistory]);
}

function usePrependAnchorRestore(options: PrependAnchorRestoreOptions) {
  useLayoutEffect(() => {
    restorePrependAnchor(options.scrollElementRef.current, options.prependAnchorRef);
  }, [
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

interface UseOlderHistoryLoadingOptions extends UseMessageListScrollOptions {
  olderLoadPendingRef: MutableRefObject<boolean>;
  prependAnchorRef: MutableRefObject<PrependAnchor | null>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
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

function capturePrependAnchor(
  container: HTMLDivElement | null,
  prependAnchorRef: MutableRefObject<PrependAnchor | null>,
) {
  if (!container) {
    return;
  }

  prependAnchorRef.current = {
    scrollHeight: container.scrollHeight,
    scrollTop: container.scrollTop,
  };
}

function restorePrependAnchor(
  container: HTMLDivElement | null,
  prependAnchorRef: MutableRefObject<PrependAnchor | null>,
) {
  const anchor = prependAnchorRef.current;
  if (!anchor || !container) {
    return;
  }

  container.scrollTop = anchor.scrollTop + (container.scrollHeight - anchor.scrollHeight);
  prependAnchorRef.current = null;
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
