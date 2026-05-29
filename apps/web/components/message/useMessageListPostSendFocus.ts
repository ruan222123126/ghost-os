import type { Virtualizer } from '@tanstack/react-virtual';
import type { MutableRefObject } from 'react';
import { useLayoutEffect, useRef, useState } from 'react';
import {
  computePostSendAnchorLayout,
  resolvePostSendOverflowDecision,
} from './messageListScroll';

const POST_SEND_TOKEN_NONE = 0;

type PostSendMode = 'idle' | 'anchoring' | 'waiting_overflow' | 'normal_follow';

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
  postSendHasVisibleAssistantText?: boolean;
  postSendToken?: number;
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
}

export function useMessageListPostSendFocus(options: UseMessageListPostSendFocusOptions) {
  const {
    autoFollowRef,
    cancelScheduledScroll,
    layoutSignature,
    loadingOlderHistory,
    onNormalLayoutChange,
    postSendHasVisibleAssistantText,
    rowVirtualizer,
    scrollElementRef,
  } = options;
  const postSendRef = useRef<PostSendState>({ mode: 'idle', token: POST_SEND_TOKEN_NONE });
  const pendingAnchorRef = useRef<PendingPostSendAnchor | null>(null);
  const pendingBottomScrollRef = useRef(false);
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
      pendingBottomScrollRef,
      postSendRef,
      layoutSignature,
      loadingOlderHistory,
      onNormalLayoutChange,
      postSendHasVisibleAssistantText,
      rowVirtualizer,
      scrollElementRef,
      setTrailingSpacerPx,
      token,
    });
  }, [
    anchorIndex,
    autoFollowRef,
    cancelScheduledScroll,
    layoutSignature,
    loadingOlderHistory,
    onNormalLayoutChange,
    postSendHasVisibleAssistantText,
    rowVirtualizer,
    scrollElementRef,
    token,
    trailingSpacerPx,
  ]);

  useLayoutEffect(() => {
    applyPendingPostSendScroll({
      pendingAnchorRef,
      pendingBottomScrollRef,
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
  pendingBottomScrollRef: MutableRefObject<boolean>;
  postSendRef: MutableRefObject<PostSendState>;
  setTrailingSpacerPx: (value: number) => void;
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
  options.postSendRef.current = { mode: 'idle', token: POST_SEND_TOKEN_NONE };
  options.pendingAnchorRef.current = null;
  options.pendingBottomScrollRef.current = false;
  options.setTrailingSpacerPx(0);
  options.onNormalLayoutChange();
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
    hasVisibleAssistantText: Boolean(options.postSendHasVisibleAssistantText),
    realContentHeightPx: options.rowVirtualizer.getTotalSize(),
  });
  if (!decision.overflowed) {
    options.postSendRef.current = { ...state, mode: 'waiting_overflow' };
    return;
  }

  options.pendingAnchorRef.current = null;
  options.pendingBottomScrollRef.current = decision.shouldScrollToBottom;
  options.setTrailingSpacerPx(decision.trailingSpacerPx ?? 0);
  options.postSendRef.current = { ...state, mode: 'normal_follow' };
}

function applyPendingPostSendScroll(options: {
  pendingAnchorRef: MutableRefObject<PendingPostSendAnchor | null>;
  pendingBottomScrollRef: MutableRefObject<boolean>;
  postSendRef: MutableRefObject<PostSendState>;
  scrollElementRef: MutableRefObject<HTMLDivElement | null>;
  trailingSpacerPx: number;
}) {
  const container = options.scrollElementRef.current;
  if (!container) {
    return;
  }

  applyPendingAnchorScroll(options, container);
  applyPendingBottomScroll(options, container);
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

  container.scrollTop = pending.scrollTopPx;
  options.pendingAnchorRef.current = null;
}

function applyPendingBottomScroll(
  options: {
    pendingBottomScrollRef: MutableRefObject<boolean>;
    trailingSpacerPx: number;
  },
  container: HTMLDivElement,
) {
  if (!options.pendingBottomScrollRef.current || options.trailingSpacerPx !== 0) {
    return;
  }

  container.scrollTop = container.scrollHeight;
  options.pendingBottomScrollRef.current = false;
}

function getPostSendAnchor(options: PostSendStateMachineOptions) {
  const historyRowOffset = options.loadingOlderHistory ? 1 : 0;
  const rowIndex = (options.anchorIndex ?? 0) + historyRowOffset;
  return options.rowVirtualizer.measurementsCache[rowIndex];
}
