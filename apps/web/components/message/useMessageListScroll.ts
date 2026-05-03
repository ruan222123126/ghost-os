import type { Virtualizer } from '@tanstack/react-virtual';
import type { MutableRefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef } from 'react';
import {
  isMessageListNearBottom,
  shouldAdjustScrollPositionOnItemSizeChange,
} from './messageListScroll';

const LOAD_OLDER_TRIGGER_ROWS = 5;

interface PrependAnchor {
  scrollHeight: number;
  scrollTop: number;
}

interface UseMessageListScrollOptions {
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  firstVirtualItemIndex: number | null;
  hasOlderHistory: boolean;
  loadingOlderHistory: boolean;
  loadOlderHistory: () => Promise<void>;
  visibleCommittedMessageCount: number;
  layoutSignature: string;
}

export function useMessageListScroll(options: UseMessageListScrollOptions) {
  const scrollElementRef = useRef<HTMLDivElement>(null);
  const autoFollowRef = useRef(true);
  const olderLoadPendingRef = useRef(false);
  const prependAnchorRef = useRef<PrependAnchor | null>(null);
  const scrollFrameRef = useRef<number | null>(null);

  const syncAutoFollow = useCallback(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    autoFollowRef.current = isMessageListNearBottom(container);
  }, []);

  const handleLoadOlderHistory = useCallback(async () => {
    if (!options.hasOlderHistory || options.loadingOlderHistory || olderLoadPendingRef.current) {
      return;
    }

    capturePrependAnchor(scrollElementRef.current, prependAnchorRef);
    olderLoadPendingRef.current = true;
    try {
      await options.loadOlderHistory();
    } finally {
      olderLoadPendingRef.current = false;
    }
  }, [options.hasOlderHistory, options.loadOlderHistory, options.loadingOlderHistory]);

  useLayoutEffect(() => {
    options.rowVirtualizer.shouldAdjustScrollPositionOnItemSizeChange = () =>
      shouldAdjustScrollPositionOnItemSizeChange(autoFollowRef.current);

    return () => {
      options.rowVirtualizer.shouldAdjustScrollPositionOnItemSizeChange = undefined;
    };
  }, [options.rowVirtualizer]);

  useEffect(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    const handleScroll = () => {
      syncAutoFollow();
    };

    handleScroll();
    container.addEventListener('scroll', handleScroll, { passive: true });
    return () => {
      container.removeEventListener('scroll', handleScroll);
    };
  }, [syncAutoFollow]);

  useEffect(() => {
    if (!options.hasOlderHistory || options.loadingOlderHistory) {
      return;
    }
    if (options.firstVirtualItemIndex === null || options.firstVirtualItemIndex > LOAD_OLDER_TRIGGER_ROWS) {
      return;
    }

    void handleLoadOlderHistory();
  }, [
    handleLoadOlderHistory,
    options.firstVirtualItemIndex,
    options.hasOlderHistory,
    options.loadingOlderHistory,
  ]);

  useLayoutEffect(() => {
    restorePrependAnchor(scrollElementRef.current, prependAnchorRef);
  }, [options.loadingOlderHistory, options.visibleCommittedMessageCount]);

  useLayoutEffect(() => {
    scheduleScrollToBottom({
      autoFollowRef,
      prependAnchorRef,
      scrollElementRef,
      scrollFrameRef,
    });
  }, [options.layoutSignature]);

  useEffect(() => {
    return () => {
      cancelScrollFrame(scrollFrameRef);
    };
  }, []);

  return { scrollElementRef };
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
    const nextContainer = options.scrollElementRef.current;
    if (!nextContainer || options.prependAnchorRef.current || !options.autoFollowRef.current) {
      return;
    }

    nextContainer.scrollTop = nextContainer.scrollHeight;
  });
}

function cancelScrollFrame(scrollFrameRef: MutableRefObject<number | null>) {
  if (scrollFrameRef.current === null) {
    return;
  }

  window.cancelAnimationFrame(scrollFrameRef.current);
  scrollFrameRef.current = null;
}
