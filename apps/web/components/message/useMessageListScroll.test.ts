import type { VirtualItem, Virtualizer } from '@tanstack/react-virtual';
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

  it('keeps the visible row anchored when older messages are prepended', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 300,
      scrollHeight: 600,
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
          rowKeys: ['message-1', 'message-2'],
          scrollElement,
          virtualItems: [
            createVirtualItem(0, 0, 100),
            createVirtualItem(1, 100, 220),
          ],
          visibleCommittedMessageCount: 2,
        }),
        {
          createNodeMock: () => scrollElement,
        },
      );
      await Promise.resolve();
    });

    await act(async () => {
      scrollElement.scrollTop = 120;
      renderer.update(
        React.createElement(HookProbe, {
          firstVirtualItemIndex: 0,
          hasOlderHistory: true,
          layoutSignature: 'session:ready',
          loadOlderHistory,
          rowKeys: ['message-1', 'message-2'],
          scrollElement,
          virtualItems: [
            createVirtualItem(0, 0, 100),
            createVirtualItem(1, 100, 220),
          ],
          visibleCommittedMessageCount: 2,
        }),
      );
      await Promise.resolve();
    });

    expect(loadOlderHistory).toHaveBeenCalledTimes(1);

    await act(async () => {
      scrollElement.scrollHeight = 900;
      renderer.update(
        React.createElement(HookProbe, {
          firstVirtualItemIndex: 8,
          firstVisibleCommittedMessageId: 'older-1',
          hasOlderHistory: true,
          layoutSignature: 'session:older-loaded',
          loadOlderHistory,
          rowKeys: ['older-1', 'older-2', 'message-1', 'message-2'],
          scrollElement,
          virtualItems: [
            createVirtualItem(0, 0, 180),
            createVirtualItem(1, 180, 400),
            createVirtualItem(2, 400, 500),
            createVirtualItem(3, 500, 620),
          ],
          visibleCommittedMessageCount: 4,
        }),
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(520);
  });

  it('shows a bottom affordance after manual upward scroll and scrolls back to the bottom', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 400,
      scrollHeight: 1600,
      scrollTop: 0,
    });
    const loadOlderHistory = jest.fn(async () => undefined);
    let latestHook: HookProbeRenderState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(HookProbe, {
          firstVirtualItemIndex: 20,
          hasOlderHistory: false,
          layoutSignature: 'session:ready',
          loadOlderHistory,
          onRender: (state) => {
            latestHook = state;
          },
          scrollElement,
          visibleCommittedMessageCount: 12,
        }),
        {
          createNodeMock: () => scrollElement,
        },
      );
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).showScrollToBottom).toBe(false);

    await act(async () => {
      scrollElement.dispatchEvent(new Event('wheel'));
      scrollElement.scrollTop = 800;
      scrollElement.dispatchEvent(new Event('scroll'));
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).showScrollToBottom).toBe(true);

    await act(async () => {
      requireLatestHook(latestHook).scrollToBottom();
      await Promise.resolve();
    });

    expect(scrollElement.scrollTo).toHaveBeenLastCalledWith({
      behavior: 'smooth',
      top: 1600,
    });
    expect(scrollElement.scrollTop).toBe(1600);
    expect(requireLatestHook(latestHook).showScrollToBottom).toBe(false);
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

    await act(async () => {
      scrollElement.dispatchEvent(new Event('wheel'));
      scrollElement.scrollTop = 250;
      scrollElement.dispatchEvent(new Event('scroll'));
      scrollElement.scrollHeight = 480;
      await Promise.resolve();
    });

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

interface HookProbeRenderState {
  scrollToBottom: () => void;
  showScrollToBottom: boolean;
}

function requireLatestHook(state: HookProbeRenderState | null): HookProbeRenderState {
  if (!state) {
    throw new Error('Hook state was not rendered');
  }

  return state;
}

function HookProbe(props: {
  firstVisibleCommittedMessageId?: string;
  firstVirtualItemIndex: number | null;
  hasOlderHistory: boolean;
  layoutSignature: string;
  loadingOlderHistory?: boolean;
  loadOlderHistory: () => Promise<void>;
  onRender?: (state: HookProbeRenderState) => void;
  postSendAnchorIndex?: number | null;
  postSendHasVisibleContent?: boolean;
  postSendToken?: number;
  rowKeys?: string[];
  rowStartPx?: number;
  scrollElement: ScrollElement;
  virtualItems?: VirtualItem[];
  visibleCommittedMessageCount: number;
}) {
  const hook = useMessageListScroll({
    firstVirtualItemIndex: props.firstVirtualItemIndex,
    firstVisibleCommittedMessageId: props.firstVisibleCommittedMessageId ?? 'message-1',
    hasOlderHistory: props.hasOlderHistory,
    layoutSignature: props.layoutSignature,
    loadOlderHistory: props.loadOlderHistory,
    loadingOlderHistory: props.loadingOlderHistory ?? false,
    postSendAnchorIndex: props.postSendAnchorIndex,
    postSendHasVisibleContent: props.postSendHasVisibleContent,
    postSendToken: props.postSendToken,
    rowKeys: props.rowKeys ?? ['message-1'],
    rowVirtualizer: createRowVirtualizer(props.scrollElement, {
      rowStartPx: props.rowStartPx,
      virtualItems: props.virtualItems,
    }),
    visibleCommittedMessageCount: props.visibleCommittedMessageCount,
  });
  props.onRender?.({
    scrollToBottom: hook.scrollToBottom,
    showScrollToBottom: hook.showScrollToBottom,
  });

  return React.createElement('div', { ref: hook.scrollElementRef });
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
    scrollTo: jest.fn((options) => {
      element.scrollTop = Number(options.top ?? element.scrollTop);
    }),
  };

  return element;
}

function createRowVirtualizer(
  scrollElement: ScrollElement,
  options: {
    rowStartPx?: number;
    virtualItems?: VirtualItem[];
  } = {},
): Virtualizer<HTMLDivElement, Element> {
  const measurementsCache = options.virtualItems
    ?? (options.rowStartPx === undefined ? [] : [createVirtualItem(0, options.rowStartPx, options.rowStartPx + 96)]);

  return {
    getOffsetForIndex: (index: number) => {
      const item = measurementsCache[index];
      return item ? [item.start, 'start'] as const : undefined;
    },
    getTotalSize: () => scrollElement.scrollHeight,
    getVirtualItems: () => measurementsCache,
    measurementsCache,
  } as unknown as Virtualizer<HTMLDivElement, Element>;
}

function createVirtualItem(index: number, start: number, end: number): VirtualItem {
  return {
    end,
    index,
    key: index,
    lane: 0,
    size: end - start,
    start,
  };
}
