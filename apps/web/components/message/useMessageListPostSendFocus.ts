import type { Virtualizer } from '@tanstack/react-virtual';
import type { MutableRefObject } from 'react';
import { useLayoutEffect, useRef, useState } from 'react';
import {
  computePostSendAnchorLayout,
  type PostSendFollowTrackingState,
  resolvePostSendOverflowDecision,
} from './messageListScroll';

const POST_SEND_TOKEN_NONE = 0;

type PostSendMode = 'idle' | 'anchoring' | 'waiting_overflow';

type PostSendState =
  | { mode: 'idle'; token: number }
  | {
    mode: Exclude<PostSendMode, 'idle'>;
    token: number;
    anchorStartPx: number;
    baselineContentHeightPx: number;
  };

interface PendingPostSendAnchor {
  requiredSpacerPx: number;
  scrollTopPx: number;
  token: number;
}

interface UseMessageListPostSendFocusOptions {
  autoFollowRef: MutableRefObject<boolean>;
  cancelScheduledScroll: () => void;
  layoutSignature: string;
  loadingOlderHistory: boolean;
  onNormalLayoutChange: () => void;
  postSendAnchorIndex?: number | null;
  postSendHasVisibleContent?: boolean;
  postSendToken?: number;
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  setPostSendFollowTracking: (value: PostSendFollowTrackingState) => void;
}

export function useMessageListPostSendFocus(options: UseMessageListPostSendFocusOptions) {
  const {
    autoFollowRef,
    cancelScheduledScroll,
    layoutSignature,
    loadingOlderHistory,
    onNormalLayoutChange,
    postSendHasVisibleContent,
    rowVirtualizer,
    setPostSendFollowTracking,
    scrollElementRef,
  } = options;
  const postSendRef = useRef<PostSendState>({ mode: 'idle', token: POST_SEND_TOKEN_NONE });
  const pendingAnchorRef = useRef<PendingPostSendAnchor | null>(null);
  const [trailingSpacerPx, setTrailingSpacerPx] = useState(0);
  const token = options.postSendToken ?? POST_SEND_TOKEN_NONE;
  const anchorIndex = options.postSendAnchorIndex ?? null;
  const postSendSignature = `${token}:${anchorIndex ?? 'none'}`;

  useLayoutEffect(() => {
    runPostSendStateMachine({
      anchorIndex,
      autoFollowRef,
      cancelScheduledScroll,
      pendingAnchorRef,
      postSendRef,
      layoutSignature,
      loadingOlderHistory,
      onNormalLayoutChange,
      postSendHasVisibleContent,
      rowVirtualizer,
      setPostSendFollowTracking,
      scrollElementRef,
      setTrailingSpacerPx,
      trailingSpacerPx,
      token,
    });
  }, [
    anchorIndex,
    autoFollowRef,
    cancelScheduledScroll,
    layoutSignature,
    loadingOlderHistory,
    onNormalLayoutChange,
    postSendHasVisibleContent,
    rowVirtualizer,
    setPostSendFollowTracking,
    scrollElementRef,
    token,
    trailingSpacerPx,
  ]);

  useLayoutEffect(() => {
    applyPendingPostSendScroll({
      pendingAnchorRef,
      postSendRef,
      scrollElementRef,
      trailingSpacerPx,
    });
  }, [postSendSignature, scrollElementRef, trailingSpacerPx]);

  return { trailingSpacerPx };
}

interface PostSendStateMachineOptions extends UseMessageListPostSendFocusOptions {
  anchorIndex: number | null;
  pendingAnchorRef: MutableRefObject<PendingPostSendAnchor | null>;
  postSendRef: MutableRefObject<PostSendState>;
  setTrailingSpacerPx: (value: number) => void;
  trailingSpacerPx: number;
  token: number;
}

function runPostSendStateMachine(options: PostSendStateMachineOptions) {
  if (options.token === POST_SEND_TOKEN_NONE || options.anchorIndex === null) {
    resetPostSendFocus(options);
    return;
  }
  if (options.postSendRef.current.token !== options.token) {
    startPostSendAnchoring(options);
    return;
  }

  continuePostSendFocus(options);
}

function resetPostSendFocus(options: PostSendStateMachineOptions) {
  const state = options.postSendRef.current;
  if (state.mode === 'idle') {
    return;
  }

  const retainedSpacerPx = options.trailingSpacerPx;
  releasePostSendFocus(options, {
    nextToken: POST_SEND_TOKEN_NONE,
    shouldScrollToBottom: options.autoFollowRef.current && retainedSpacerPx === 0,
    trailingSpacerPx: retainedSpacerPx,
  });
}

function startPostSendAnchoring(options: PostSendStateMachineOptions) {
  const container = options.scrollElementRef.current;
  if (!container) {
    return;
  }

  options.cancelScheduledScroll();
  options.autoFollowRef.current = true;
  const realContentHeightPx = options.rowVirtualizer.getTotalSize();
  const anchor = getPostSendAnchor(options);
  if (!anchor) {
    throw new Error('post-send anchor row is unavailable');
  }
  const layout = computePostSendAnchorLayout({
    anchorStartPx: anchor.start,
    containerHeightPx: container.clientHeight,
    realContentHeightPx,
  });
  options.setTrailingSpacerPx(layout.trailingSpacerPx);
  options.setPostSendFollowTracking({
    mode: 'anchoring',
    controlledScrollTopPx: layout.anchorScrollTopPx,
    programmaticScrollTargetPx: layout.anchorScrollTopPx,
  });
  options.pendingAnchorRef.current = {
    requiredSpacerPx: layout.trailingSpacerPx,
    scrollTopPx: layout.anchorScrollTopPx,
    token: options.token,
  };
  options.postSendRef.current = {
    mode: 'anchoring',
    token: options.token,
    anchorStartPx: anchor.start,
    baselineContentHeightPx: realContentHeightPx,
  };
}

function continuePostSendFocus(options: PostSendStateMachineOptions) {
  const container = options.scrollElementRef.current;
  const state = options.postSendRef.current;
  if (!container || state.mode === 'idle') {
    return;
  }

  const decision = resolvePostSendOverflowDecision({
    anchorStartPx: state.anchorStartPx,
    autoFollow: options.autoFollowRef.current,
    baselineContentHeightPx: state.baselineContentHeightPx,
    containerHeightPx: container.clientHeight,
    hasVisibleContent: Boolean(options.postSendHasVisibleContent),
    realContentHeightPx: options.rowVirtualizer.getTotalSize(),
  });
  if (!decision.overflowed) {
    options.setPostSendFollowTracking({
      mode: 'waiting_overflow',
      controlledScrollTopPx: state.anchorStartPx,
      programmaticScrollTargetPx: state.anchorStartPx,
    });
    options.pendingAnchorRef.current = {
      requiredSpacerPx: decision.trailingSpacerPx,
      scrollTopPx: state.anchorStartPx,
      token: options.token,
    };
    options.setTrailingSpacerPx(decision.trailingSpacerPx);
    options.postSendRef.current = { ...state, mode: 'waiting_overflow' };
    return;
  }

  releasePostSendFocus(options, {
    nextToken: options.token,
    shouldScrollToBottom: decision.shouldScrollToBottom,
  });
}

function releasePostSendFocus(
  options: PostSendStateMachineOptions,
  release: {
    nextToken: number;
    shouldScrollToBottom: boolean;
    trailingSpacerPx?: number;
  },
) {
  options.postSendRef.current = { mode: 'idle', token: release.nextToken };
  options.pendingAnchorRef.current = null;
  options.setPostSendFollowTracking({
    mode: 'idle',
    controlledScrollTopPx: null,
    programmaticScrollTargetPx: null,
  });
  options.setTrailingSpacerPx(release.trailingSpacerPx ?? 0);
  if (release.shouldScrollToBottom) {
    options.onNormalLayoutChange();
  }
}

function applyPendingPostSendScroll(options: {
  pendingAnchorRef: MutableRefObject<PendingPostSendAnchor | null>;
  postSendRef: MutableRefObject<PostSendState>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  trailingSpacerPx: number;
}) {
  const container = options.scrollElementRef.current;
  if (!container) {
    return;
  }

  applyPendingAnchorScroll(options, container);
}

function applyPendingAnchorScroll(
  options: {
    pendingAnchorRef: MutableRefObject<PendingPostSendAnchor | null>;
    postSendRef: MutableRefObject<PostSendState>;
    trailingSpacerPx: number;
  },
  container: HTMLDivElement,
) {
  const pending = options.pendingAnchorRef.current;
  if (!pending || options.postSendRef.current.token !== pending.token) {
    return;
  }
  if (pending.requiredSpacerPx !== options.trailingSpacerPx) {
    return;
  }

  container.scrollTo({
    top: pending.scrollTopPx,
  });
  options.pendingAnchorRef.current = null;
}

function getPostSendAnchor(options: PostSendStateMachineOptions) {
  const historyRowOffset = options.loadingOlderHistory ? 1 : 0;
  const rowIndex = (options.anchorIndex ?? 0) + historyRowOffset;
  return options.rowVirtualizer.measurementsCache[rowIndex];
}
