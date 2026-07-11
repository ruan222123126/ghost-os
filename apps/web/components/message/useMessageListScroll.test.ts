import type { Virtualizer } from '@tanstack/react-virtual';
import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { useMessageListScroll } from './useMessageListScroll';

describe('components/message/useMessageListScroll', () => {
  it('starts loaded history at the bottom without immediately loading older messages', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 400,
      scrollHeight: 1600,
      scrollTop: 0,
    });
    const loadOlderHistory = jest.fn(async () => undefined);

    await act(async () => {
      TestRenderer.create(
        React.createElement(HookProbe, {
          firstVirtualItemIndex: 0,
          hasOlderHistory: true,
          layoutSignature: 'session:ready',
          loadOlderHistory,
          scrollElement,
          visibleCommittedMessageCount: 12,
        }),
        {
          createNodeMock: () => scrollElement,
        },
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(1600);
    expect(loadOlderHistory).not.toHaveBeenCalled();
  });

  it('loads older messages after the user reaches the top following initial placement', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 400,
      scrollHeight: 1600,
      scrollTop: 0,
    });
    const loadOlderHistory = jest.fn(async () => undefined);
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          firstVirtualItemIndex: 20,
          hasOlderHistory: true,
          layoutSignature: 'session:ready',
          loadOlderHistory,
          scrollElement,
          visibleCommittedMessageCount: 12,
        }),
        {
          createNodeMock: () => scrollElement,
        },
      );
      await Promise.resolve();
    });

    expect(loadOlderHistory).not.toHaveBeenCalled();

    await act(async () => {
      scrollElement.scrollTop = 0;
      renderer.update(
        React.createElement(HookProbe, {
          firstVirtualItemIndex: 0,
          hasOlderHistory: true,
          layoutSignature: 'session:ready',
          loadOlderHistory,
          scrollElement,
          visibleCommittedMessageCount: 12,
        }),
      );
      await Promise.resolve();
    });

    expect(loadOlderHistory).toHaveBeenCalledTimes(1);
  });

  it('keeps the post-send anchor after an automatic layout scroll', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 400,
      scrollHeight: 500,
      scrollTop: 0,
    });
    const loadOlderHistory = jest.fn(async () => undefined);
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          firstVirtualItemIndex: null,
          hasOlderHistory: false,
          layoutSignature: 'post-send:start',
          loadOlderHistory,
          postSendAnchorIndex: 0,
          postSendHasVisibleContent: false,
          postSendToken: 1,
          rowStartPx: 300,
          scrollElement,
          visibleCommittedMessageCount: 0,
        }),
        {
          createNodeMock: () => scrollElement,
        },
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(300);

    scrollElement.scrollHeight = 480;
    scrollElement.scrollTop = 250;
    scrollElement.dispatchEvent(new Event('scroll'));

    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          firstVirtualItemIndex: null,
          hasOlderHistory: false,
          layoutSignature: 'post-send:thinking-collapsed',
          loadOlderHistory,
          postSendAnchorIndex: 0,
          postSendHasVisibleContent: false,
          postSendToken: 1,
          rowStartPx: 300,
          scrollElement,
          visibleCommittedMessageCount: 0,
        }),
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(300);
  });

  it('releases the post-send anchor after a manual wheel scroll', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 400,
      scrollHeight: 500,
      scrollTop: 0,
    });
    const loadOlderHistory = jest.fn(async () => undefined);
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          firstVirtualItemIndex: null,
          hasOlderHistory: false,
          layoutSignature: 'post-send:start',
          loadOlderHistory,
          postSendAnchorIndex: 0,
          postSendHasVisibleContent: false,
          postSendToken: 1,
          rowStartPx: 300,
          scrollElement,
          visibleCommittedMessageCount: 0,
        }),
        {
          createNodeMock: () => scrollElement,
        },
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(300);

    scrollElement.dispatchEvent(new Event('wheel'));
    scrollElement.scrollTop = 250;
    scrollElement.dispatchEvent(new Event('scroll'));
    scrollElement.scrollHeight = 480;

    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          firstVirtualItemIndex: null,
          hasOlderHistory: false,
          layoutSignature: 'post-send:manual-scroll',
          loadOlderHistory,
          postSendAnchorIndex: 0,
          postSendHasVisibleContent: false,
          postSendToken: 1,
          rowStartPx: 300,
          scrollElement,
          visibleCommittedMessageCount: 0,
        }),
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(250);
  });
});

function HookProbe(props: {
  firstVirtualItemIndex: number | null;
  hasOlderHistory: boolean;
  layoutSignature: string;
  loadOlderHistory: () => Promise<void>;
  postSendAnchorIndex?: number | null;
  postSendHasVisibleContent?: boolean;
  postSendToken?: number;
  rowStartPx?: number;
  scrollElement: ScrollElement;
  visibleCommittedMessageCount: number;
}) {
  const { scrollElementRef } = useMessageListScroll({
    firstVirtualItemIndex: props.firstVirtualItemIndex,
    firstVisibleCommittedMessageId: 'message-1',
    hasOlderHistory: props.hasOlderHistory,
    layoutSignature: props.layoutSignature,
    loadOlderHistory: props.loadOlderHistory,
    loadingOlderHistory: false,
    postSendAnchorIndex: props.postSendAnchorIndex,
    postSendHasVisibleContent: props.postSendHasVisibleContent,
    postSendToken: props.postSendToken,
    rowVirtualizer: createRowVirtualizer(props.scrollElement, props.rowStartPx),
    visibleCommittedMessageCount: props.visibleCommittedMessageCount,
  });

  return React.createElement('div', { ref: scrollElementRef });
}

interface ScrollElement {
  addEventListener: (type: string, listener: EventListener) => void;
  clientHeight: number;
  dispatchEvent: (event: Event) => boolean;
  removeEventListener: (type: string, listener: EventListener) => void;
  scrollHeight: number;
  scrollTo: (options: ScrollToOptions) => void;
  scrollTop: number;
}

function createScrollElement(layout: {
  clientHeight: number;
  scrollHeight: number;
  scrollTop: number;
}): ScrollElement {
  const listeners = new Map<string, Set<EventListener>>();
  const element: ScrollElement = {
    ...layout,
    addEventListener: (type, listener) => {
      const current = listeners.get(type) ?? new Set<EventListener>();
      current.add(listener);
      listeners.set(type, current);
    },
    dispatchEvent: (event) => {
      for (const listener of listeners.get(event.type) ?? []) {
        listener(event);
      }
      return true;
    },
    removeEventListener: (type, listener) => {
      listeners.get(type)?.delete(listener);
    },
    scrollTo: (options) => {
      element.scrollTop = Number(options.top ?? element.scrollTop);
    },
  };

  return element;
}

function createRowVirtualizer(
  scrollElement: ScrollElement,
  rowStartPx?: number,
): Virtualizer<HTMLDivElement, Element> {
  return {
    getTotalSize: () => scrollElement.scrollHeight,
    measurementsCache: rowStartPx === undefined ? [] : [{ start: rowStartPx }],
  } as unknown as Virtualizer<HTMLDivElement, Element>;
}
