import type { Virtualizer } from '@tanstack/react-virtual';
import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import type { PostSendFollowTrackingState } from './messageListScroll';
import { useMessageListPostSendFocus } from './useMessageListPostSendFocus';

describe('components/message/useMessageListPostSendFocus', () => {
  it('releases the post-send lock once streamed content fills the anchored viewport', async () => {
    const autoFollowRef = { current: true };
    const scrollElementRef = createScrollRef({ clientHeight: 400, scrollTop: 0 });
    const onNormalLayoutChange = jest.fn();
    const cancelScheduledScroll = jest.fn();
    const trackingStates: PostSendFollowTrackingState[] = [];
    let trailingSpacerPx = 0;
    let totalSize = 500;
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          autoFollowRef,
          cancelScheduledScroll,
          layoutSignature: 'start',
          onNormalLayoutChange,
          onRender: (value) => {
            trailingSpacerPx = value;
          },
          postSendHasVisibleContent: false,
          rowVirtualizer: createRowVirtualizer(() => totalSize),
          scrollElementRef,
          setPostSendFollowTracking: (value) => {
            trackingStates.push(value);
          },
        }),
      );
      await Promise.resolve();
    });

    expect(trailingSpacerPx).toBe(200);
    expect(scrollElementRef.current?.scrollTop).toBe(300);
    expect(getScrollToMock(scrollElementRef)).toHaveBeenCalledWith({
      behavior: 'smooth',
      top: 300,
    });
    expect(trackingStates.at(-1)).toEqual({
      mode: 'waiting_overflow',
      controlledScrollTopPx: 300,
      programmaticScrollTargetPx: 300,
    });

    totalSize = 701;
    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          autoFollowRef,
          cancelScheduledScroll,
          layoutSignature: 'overflowed',
          onNormalLayoutChange,
          onRender: (value) => {
            trailingSpacerPx = value;
          },
          postSendHasVisibleContent: true,
          rowVirtualizer: createRowVirtualizer(() => totalSize),
          scrollElementRef,
          setPostSendFollowTracking: (value) => {
            trackingStates.push(value);
          },
        }),
      );
      await Promise.resolve();
    });

    expect(trailingSpacerPx).toBe(0);
    expect(trackingStates.at(-1)).toEqual({
      mode: 'idle',
      controlledScrollTopPx: null,
      programmaticScrollTargetPx: null,
    });
    expect(onNormalLayoutChange).toHaveBeenCalledTimes(1);

    totalSize = 740;
    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          autoFollowRef,
          cancelScheduledScroll,
          layoutSignature: 'overflowed-more',
          onNormalLayoutChange,
          onRender: (value) => {
            trailingSpacerPx = value;
          },
          postSendHasVisibleContent: true,
          rowVirtualizer: createRowVirtualizer(() => totalSize),
          scrollElementRef,
          setPostSendFollowTracking: (value) => {
            trackingStates.push(value);
          },
        }),
      );
      await Promise.resolve();
    });

    expect(trailingSpacerPx).toBe(0);
    expect(onNormalLayoutChange).toHaveBeenCalledTimes(1);
  });

  it('keeps the remaining spacer when the turn ends before overflowing the anchored viewport', async () => {
    const autoFollowRef = { current: true };
    const scrollElementRef = createScrollRef({ clientHeight: 400, scrollTop: 0 });
    const onNormalLayoutChange = jest.fn();
    const trackingStates: PostSendFollowTrackingState[] = [];
    let trailingSpacerPx = 0;
    const totalSize = 500;
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          autoFollowRef,
          cancelScheduledScroll: jest.fn(),
          layoutSignature: 'start',
          onNormalLayoutChange,
          onRender: (value) => {
            trailingSpacerPx = value;
          },
          postSendAnchorIndex: 0,
          postSendHasVisibleContent: false,
          postSendToken: 1,
          rowVirtualizer: createRowVirtualizer(() => totalSize),
          scrollElementRef,
          setPostSendFollowTracking: (value) => {
            trackingStates.push(value);
          },
        }),
      );
      await Promise.resolve();
    });

    expect(trailingSpacerPx).toBe(200);

    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          autoFollowRef,
          cancelScheduledScroll: jest.fn(),
          layoutSignature: 'turn-ended',
          onNormalLayoutChange,
          onRender: (value) => {
            trailingSpacerPx = value;
          },
          postSendAnchorIndex: null,
          postSendHasVisibleContent: true,
          postSendToken: 1,
          rowVirtualizer: createRowVirtualizer(() => totalSize),
          scrollElementRef,
          setPostSendFollowTracking: (value) => {
            trackingStates.push(value);
          },
        }),
      );
      await Promise.resolve();
    });

    expect(trailingSpacerPx).toBe(200);
    expect(trackingStates.at(-1)).toEqual({
      mode: 'idle',
      controlledScrollTopPx: null,
      programmaticScrollTargetPx: null,
    });
    expect(onNormalLayoutChange).not.toHaveBeenCalled();
  });
});

function HookProbe(props: {
  autoFollowRef: React.MutableRefObject<boolean>;
  cancelScheduledScroll: () => void;
  layoutSignature: string;
  onNormalLayoutChange: () => void;
  onRender: (trailingSpacerPx: number) => void;
  postSendAnchorIndex?: number | null;
  postSendHasVisibleContent: boolean;
  postSendToken?: number;
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  scrollElementRef: React.MutableRefObject<HTMLDivElement | null>;
  setPostSendFollowTracking: (value: PostSendFollowTrackingState) => void;
}) {
  const { trailingSpacerPx } = useMessageListPostSendFocus({
    autoFollowRef: props.autoFollowRef,
    cancelScheduledScroll: props.cancelScheduledScroll,
    layoutSignature: props.layoutSignature,
    loadingOlderHistory: false,
    onNormalLayoutChange: props.onNormalLayoutChange,
    postSendAnchorIndex: props.postSendAnchorIndex === undefined ? 0 : props.postSendAnchorIndex,
    postSendHasVisibleContent: props.postSendHasVisibleContent,
    postSendToken: props.postSendToken ?? 1,
    rowVirtualizer: props.rowVirtualizer,
    scrollElementRef: props.scrollElementRef,
    setPostSendFollowTracking: props.setPostSendFollowTracking,
  });
  props.onRender(trailingSpacerPx);
  return null;
}

function createRowVirtualizer(getTotalSize: () => number) {
  return {
    getTotalSize,
    measurementsCache: [{ start: 300 }],
  } as unknown as Virtualizer<HTMLDivElement, Element>;
}

function createScrollRef(layout: {
  clientHeight: number;
  scrollTop: number;
}): React.MutableRefObject<HTMLDivElement | null> {
  const element = {
    ...layout,
    scrollTo: jest.fn((options: ScrollToOptions) => {
      element.scrollTop = Number(options.top ?? element.scrollTop);
    }),
  };

  return {
    current: element as unknown as HTMLDivElement,
  };
}

function getScrollToMock(
  scrollElementRef: React.MutableRefObject<HTMLDivElement | null>,
): jest.Mock {
  return scrollElementRef.current?.scrollTo as unknown as jest.Mock;
}
