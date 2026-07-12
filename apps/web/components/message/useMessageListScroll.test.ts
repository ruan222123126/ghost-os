import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { useMessageListScroll } from './useMessageListScroll';

const EMPTY_VISIBLE_MESSAGES: [] = [];

describe('components/message/useMessageListScroll', () => {
  afterEach(() => {
    jest.useRealTimers();
  });

  it('starts standard-flow history at the latest message bottom without loading older messages', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 400,
      scrollHeight: 1600,
      scrollTop: 0,
    });
    const loadOlderHistory = jest.fn(async () => undefined);

    await act(async () => {
      TestRenderer.create(
        React.createElement(HookProbe, {
          hasOlderHistory: true,
          layoutSignature: 'session:ready',
          loadOlderHistory,
          rowCount: 12,
          scrollElement,
        }),
        {
          createNodeMock: createNodeMock(scrollElement),
        },
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(1200);
    expect(loadOlderHistory).not.toHaveBeenCalled();
  });

  it('loads older messages near the top and compensates prepended height', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 300,
      scrollHeight: 900,
      scrollTop: 0,
    });
    const loadOlderHistory = jest.fn(async () => undefined);
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          hasOlderHistory: true,
          layoutSignature: 'session:ready',
          loadOlderHistory,
          rowCount: 12,
          scrollElement,
        }),
        {
          createNodeMock: createNodeMock(scrollElement),
        },
      );
      await Promise.resolve();
    });

    await act(async () => {
      scrollElement.scrollTop = 100;
      scrollElement.dispatchEvent(new Event('scroll'));
      await Promise.resolve();
    });

    expect(loadOlderHistory).toHaveBeenCalledTimes(1);

    scrollElement.scrollHeight = 1200;
    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          hasOlderHistory: true,
          layoutSignature: 'session:older-loaded',
          loadOlderHistory,
          rowCount: 22,
          scrollElement,
        }),
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(400);
  });

  it('shows a bottom affordance after upward scroll and scrolls back to the bottom', async () => {
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
          hasOlderHistory: false,
          layoutSignature: 'session:ready',
          loadOlderHistory,
          onRender: (state) => {
            latestHook = state;
          },
          rowCount: 12,
          scrollElement,
        }),
        {
          createNodeMock: createNodeMock(scrollElement),
        },
      );
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).showScrollToBottom).toBe(false);

    await act(async () => {
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
      top: 1200,
    });
    expect(scrollElement.scrollTop).toBe(1200);
    expect(requireLatestHook(latestHook).showScrollToBottom).toBe(false);
  });

  it('focuses the requested user message and adds standard-flow trailing space', async () => {
    jest.useFakeTimers();
    const scrollElement = createScrollElement({
      clientHeight: 500,
      scrollHeight: 300,
      scrollTop: 0,
    });
    const userRowElement = createMeasuredElement(() => 120 - scrollElement.scrollTop);
    const loadOlderHistory = jest.fn(async () => undefined);
    let latestHook: HookProbeRenderState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:post-send',
          loadOlderHistory,
          onRender: (state) => {
            scrollElement.scrollHeight = 300 + state.trailingSpacerPx;
            latestHook = state;
          },
          postSendFocusRequest: { messageId: 'user-1', token: 1 },
          rowCount: 1,
          scrollElement,
          userRowElement,
          visibleCommittedMessages: [{ id: 'user-1', kind: 'user', content: 'hello' }],
        }),
        {
          createNodeMock: createNodeMock(scrollElement, userRowElement),
        },
      );
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(288);

    await act(async () => {
      jest.runOnlyPendingTimers();
      await Promise.resolve();
    });

    expect(scrollElement.scrollTo).toHaveBeenLastCalledWith({
      behavior: 'auto',
      top: 88,
    });
  });

  it('stops auto-follow immediately when the user scrolls upward near the bottom', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 400,
      scrollHeight: 1600,
      scrollTop: 0,
    });
    const loadOlderHistory = jest.fn(async () => undefined);
    let latestHook: HookProbeRenderState | null = null;
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:ready',
          loadOlderHistory,
          onRender: (state) => {
            latestHook = state;
          },
          rowCount: 12,
          scrollElement,
        }),
        {
          createNodeMock: createNodeMock(scrollElement),
        },
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(1200);

    await act(async () => {
      scrollElement.scrollTop = 1190;
      scrollElement.dispatchEvent(new Event('scroll'));
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).showScrollToBottom).toBe(true);

    scrollElement.scrollHeight = 1700;
    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:streaming-growth',
          loadOlderHistory,
          onRender: (state) => {
            latestHook = state;
          },
          rowCount: 12,
          scrollElement,
        }),
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(1190);
  });

  it('does not follow streaming layout changes while the user scroll lock is active', async () => {
    jest.useFakeTimers();
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
          hasOlderHistory: false,
          layoutSignature: 'session:ready',
          loadOlderHistory,
          rowCount: 12,
          scrollElement,
        }),
        {
          createNodeMock: createNodeMock(scrollElement),
        },
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(1200);

    await act(async () => {
      scrollElement.dispatchEvent(new Event('wheel'));
      scrollElement.scrollHeight = 1700;
      renderer.update(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:streaming-growth',
          loadOlderHistory,
          rowCount: 12,
          scrollElement,
        }),
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(1200);

    await act(async () => {
      jest.advanceTimersByTime(1500);
      jest.runOnlyPendingTimers();
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(1300);
  });

  it('reserves post-send viewport space while the requested user row is not virtualized yet', async () => {
    jest.useFakeTimers();
    const scrollElement = createScrollElement({
      clientHeight: 500,
      scrollHeight: 3000,
      scrollTop: 900,
    });
    const loadOlderHistory = jest.fn(async () => undefined);
    let latestHook: HookProbeRenderState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:post-send:pending-row',
          loadOlderHistory,
          onRender: (state) => {
            scrollElement.scrollHeight = 3000 + state.trailingSpacerPx;
            latestHook = state;
          },
          postSendFocusRequest: { messageId: 'user-latest', token: 1 },
          rowCount: 20,
          scrollElement,
          visibleCommittedMessages: [{ id: 'user-latest', kind: 'user', content: 'latest question' }],
        }),
        {
          createNodeMock: createNodeMock(scrollElement),
        },
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTo).not.toHaveBeenCalled();
    expect(requireLatestHook(latestHook).showScrollToBottom).toBe(false);
    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(468);
  });

  it('clears previous post-send space before reserving a new turn', async () => {
    jest.useFakeTimers();
    const scrollElement = createScrollElement({
      clientHeight: 500,
      scrollHeight: 3000,
      scrollTop: 900,
    });
    const loadOlderHistory = jest.fn(async () => undefined);
    let latestHook: HookProbeRenderState | null = null;
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:post-send:first-pending-row',
          loadOlderHistory,
          onRender: (state) => {
            scrollElement.scrollHeight = 3000 + state.trailingSpacerPx;
            latestHook = state;
          },
          postSendFocusRequest: { messageId: 'user-first', token: 1 },
          rowCount: 20,
          scrollElement,
          visibleCommittedMessages: [{ id: 'user-first', kind: 'user', content: 'first question' }],
        }),
        {
          createNodeMock: createNodeMock(scrollElement),
        },
      );
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(468);

    scrollElement.clientHeight = 200;
    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:post-send:second-pending-row',
          loadOlderHistory,
          onRender: (state) => {
            scrollElement.scrollHeight = 3000 + state.trailingSpacerPx;
            latestHook = state;
          },
          postSendFocusRequest: { messageId: 'user-second', token: 2 },
          rowCount: 21,
          scrollElement,
          visibleCommittedMessages: [{ id: 'user-second', kind: 'user', content: 'second question' }],
        }),
      );
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(168);
  });

  it('consumes trailing space as streamed content fills the reserved viewport', async () => {
    jest.useFakeTimers();
    let realContentHeight = 420;
    const scrollElement = createScrollElement({
      clientHeight: 500,
      scrollHeight: realContentHeight,
      scrollTop: 0,
    });
    const latestUserRow = createMeasuredElement(() => 120 - scrollElement.scrollTop);
    const loadOlderHistory = jest.fn(async () => undefined);
    const messages = [
      { id: 'user-latest', kind: 'user' as const, content: 'latest question' },
    ];
    let latestHook: HookProbeRenderState | null = null;
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:post-send',
          loadOlderHistory,
          onRender: (state) => {
            scrollElement.scrollHeight = realContentHeight + state.trailingSpacerPx;
            latestHook = state;
          },
          postSendFocusRequest: { messageId: 'user-latest', token: 1 },
          rowCount: 1,
          scrollElement,
          userRows: {
            'user-latest': latestUserRow,
          },
          visibleCommittedMessages: messages,
        }),
        {
          createNodeMock: createNodeMock(scrollElement, {
            'user-latest': latestUserRow,
          }),
        },
      );
      await Promise.resolve();
    });

    await act(async () => {
      jest.runOnlyPendingTimers();
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(168);

    scrollElement.scrollTo = jest.fn((options) => {
      scrollElement.scrollTop = Number(options.top ?? scrollElement.scrollTop);
    });
    scrollElement.scrollTop = 88;
    realContentHeight = 500;

    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:post-send:assistant-grew',
          loadOlderHistory,
          onRender: (state) => {
            scrollElement.scrollHeight = realContentHeight + state.trailingSpacerPx;
            latestHook = state;
          },
          postSendFocusRequest: { messageId: 'user-latest', token: 1 },
          rowCount: 1,
          scrollElement,
          userRows: {
            'user-latest': latestUserRow,
          },
          visibleCommittedMessages: messages,
        }),
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTo).not.toHaveBeenCalled();
    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(88);
  });

  it('keeps remaining post-send space after a short response is committed', async () => {
    jest.useFakeTimers();
    let realContentHeight = 420;
    const scrollElement = createScrollElement({
      clientHeight: 500,
      scrollHeight: realContentHeight,
      scrollTop: 0,
    });
    const latestUserRow = createMeasuredElement(() => 120 - scrollElement.scrollTop);
    const loadOlderHistory = jest.fn(async () => undefined);
    const messages = [
      { id: 'user-latest', kind: 'user' as const, content: 'latest question' },
    ];
    let latestHook: HookProbeRenderState | null = null;
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:post-send',
          loadOlderHistory,
          onRender: (state) => {
            scrollElement.scrollHeight = realContentHeight + state.trailingSpacerPx;
            latestHook = state;
          },
          postSendFocusRequest: { messageId: 'user-latest', token: 1 },
          rowCount: 1,
          scrollElement,
          userRows: {
            'user-latest': latestUserRow,
          },
          visibleCommittedMessages: messages,
        }),
        {
          createNodeMock: createNodeMock(scrollElement, {
            'user-latest': latestUserRow,
          }),
        },
      );
      await Promise.resolve();
    });

    await act(async () => {
      jest.runOnlyPendingTimers();
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(168);

    scrollElement.scrollTo = jest.fn((options) => {
      scrollElement.scrollTop = Number(options.top ?? scrollElement.scrollTop);
    });
    scrollElement.scrollTop = 88;
    realContentHeight = 470;

    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:post-send:assistant-committed-short',
          loadOlderHistory,
          onRender: (state) => {
            scrollElement.scrollHeight = realContentHeight + state.trailingSpacerPx;
            latestHook = state;
          },
          postSendFocusRequest: { messageId: 'user-latest', token: 1 },
          rowCount: 2,
          scrollElement,
          userRows: {
            'user-latest': latestUserRow,
          },
          visibleCommittedMessages: messages,
        }),
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTo).not.toHaveBeenCalled();
    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(118);
  });

  it('keeps post-send space after user scroll and only consumes it when response content grows', async () => {
    jest.useFakeTimers();
    let realContentHeight = 300;
    const scrollElement = createScrollElement({
      clientHeight: 500,
      scrollHeight: realContentHeight,
      scrollTop: 0,
    });
    const userRowElement = createMeasuredElement(() => 120 - scrollElement.scrollTop);
    const loadOlderHistory = jest.fn(async () => undefined);
    let latestHook: HookProbeRenderState | null = null;
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:post-send',
          loadOlderHistory,
          onRender: (state) => {
            scrollElement.scrollHeight = realContentHeight + state.trailingSpacerPx;
            latestHook = state;
          },
          postSendFocusRequest: { messageId: 'user-1', token: 1 },
          rowCount: 1,
          scrollElement,
          userRowElement,
          visibleCommittedMessages: [{ id: 'user-1', kind: 'user', content: 'hello' }],
        }),
        {
          createNodeMock: createNodeMock(scrollElement, userRowElement),
        },
      );
      await Promise.resolve();
    });

    await act(async () => {
      jest.runOnlyPendingTimers();
      scrollElement.dispatchEvent(new Event('scroll'));
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(288);
    expect(requireLatestHook(latestHook).showScrollToBottom).toBe(false);

    await act(async () => {
      scrollElement.dispatchEvent(new Event('wheel'));
      scrollElement.scrollTop = 48;
      scrollElement.dispatchEvent(new Event('scroll'));
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(288);
    expect(requireLatestHook(latestHook).showScrollToBottom).toBe(true);

    scrollElement.scrollTo = jest.fn((options) => {
      scrollElement.scrollTop = Number(options.top ?? scrollElement.scrollTop);
    });
    realContentHeight = 420;

    await act(async () => {
      renderer.update(
        React.createElement(HookProbe, {
          hasOlderHistory: false,
          layoutSignature: 'session:post-send:assistant-grew-after-user-scroll',
          loadOlderHistory,
          onRender: (state) => {
            scrollElement.scrollHeight = realContentHeight + state.trailingSpacerPx;
            latestHook = state;
          },
          postSendFocusRequest: { messageId: 'user-1', token: 1 },
          rowCount: 1,
          scrollElement,
          userRowElement,
          visibleCommittedMessages: [{ id: 'user-1', kind: 'user', content: 'hello' }],
        }),
      );
      await Promise.resolve();
    });

    expect(scrollElement.scrollTo).not.toHaveBeenCalled();
    expect(requireLatestHook(latestHook).trailingSpacerPx).toBe(168);
  });
});

interface HookProbeRenderState {
  olderHistoryLoadingPaused: boolean;
  scrollToBottom: () => void;
  showScrollToBottom: boolean;
  trailingSpacerPx: number;
}

function requireLatestHook(state: HookProbeRenderState | null): HookProbeRenderState {
  if (!state) {
    throw new Error('Hook state was not rendered');
  }

  return state;
}

function HookProbe(props: {
  hasOlderHistory: boolean;
  layoutSignature: string;
  loadingOlderHistory?: boolean;
  loadOlderHistory: () => Promise<void>;
  onRender?: (state: HookProbeRenderState) => void;
  postSendFocusRequest?: { messageId: string; token: number } | null;
  rowCount: number;
  scrollElement: ScrollElement;
  userRowElement?: MeasuredElement;
  userRows?: Record<string, MeasuredElement>;
  visibleCommittedMessages?: Array<{ id: string; kind: 'user'; content: string }>;
}) {
  const hook = useMessageListScroll({
    hasOlderHistory: props.hasOlderHistory,
    layoutSignature: props.layoutSignature,
    loadOlderHistory: props.loadOlderHistory,
    loadingOlderHistory: props.loadingOlderHistory ?? false,
    postSendFocusRequest: props.postSendFocusRequest ?? null,
    rowCount: props.rowCount,
    visibleCommittedMessages: props.visibleCommittedMessages ?? EMPTY_VISIBLE_MESSAGES,
  });
  props.onRender?.({
    olderHistoryLoadingPaused: hook.olderHistoryLoadingPaused,
    scrollToBottom: hook.scrollToBottom,
    showScrollToBottom: hook.showScrollToBottom,
    trailingSpacerPx: hook.trailingSpacerPx,
  });

  return React.createElement(
    'div',
    { ref: hook.scrollElementRef, 'data-node': 'scroll' },
    props.userRows
      ? Object.keys(props.userRows).map((messageId) => (
        React.createElement('div', {
          key: messageId,
          ref: hook.registerMessageRow(messageId),
          'data-message-id': messageId,
          'data-node': 'user-row',
        })
      ))
      : null,
    props.userRowElement
      ? React.createElement('div', { ref: hook.registerMessageRow('user-1'), 'data-node': 'user-row' })
      : null,
    React.createElement('div', { ref: hook.historySentinelRef, 'data-node': 'sentinel' }),
  );
}

interface ScrollElement {
  addEventListener: (type: string, listener: EventListener) => void;
  clientHeight: number;
  dispatchEvent: (event: Event) => boolean;
  getBoundingClientRect: () => Pick<DOMRect, 'top'>;
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
    getBoundingClientRect: () => ({ top: 0 }),
    removeEventListener: (type, listener) => {
      listeners.get(type)?.delete(listener);
    },
    scrollTo: jest.fn((options) => {
      element.scrollTop = Number(options.top ?? element.scrollTop);
    }),
  };

  return element;
}

interface MeasuredElement {
  getBoundingClientRect: () => Pick<DOMRect, 'top'>;
}

function createMeasuredElement(resolveTop: () => number): MeasuredElement {
  return {
    getBoundingClientRect: () => ({ top: resolveTop() }),
  };
}

function createNodeMock(
  scrollElement: ScrollElement,
  userRowElement?: MeasuredElement | Record<string, MeasuredElement>,
) {
  return (element: React.ReactElement) => {
    if (element.props['data-node'] === 'scroll') {
      return scrollElement;
    }
    if (element.props['data-node'] === 'user-row') {
      const messageId = element.props['data-message-id'];
      if (typeof messageId === 'string' && userRowElement && !('getBoundingClientRect' in userRowElement)) {
        return userRowElement[messageId] ?? {};
      }
      return userRowElement ?? {};
    }
    if (element.props['data-node'] === 'sentinel') {
      return {};
    }

    return null;
  };
}
