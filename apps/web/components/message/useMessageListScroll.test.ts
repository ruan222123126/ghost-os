import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { useMessageListScroll } from './useMessageListScroll';

describe('components/message/useMessageListScroll', () => {
  let intersectionCallback: IntersectionObserverCallback | null = null;
  let originalIntersectionObserver: typeof IntersectionObserver | undefined;

  beforeEach(() => {
    originalIntersectionObserver = globalThis.IntersectionObserver;
    intersectionCallback = null;
    globalThis.IntersectionObserver = class MockIntersectionObserver {
      readonly root: Element | Document | null = null;
      readonly rootMargin = '0px';
      readonly thresholds = [0.1];

      constructor(callback: IntersectionObserverCallback) {
        intersectionCallback = callback;
      }

      disconnect() {}
      observe() {}
      takeRecords(): IntersectionObserverEntry[] {
        return [];
      }
      unobserve() {}
    };
  });

  afterEach(() => {
    jest.useRealTimers();
    if (originalIntersectionObserver) {
      globalThis.IntersectionObserver = originalIntersectionObserver;
      return;
    }

    delete (globalThis as Partial<typeof globalThis>).IntersectionObserver;
  });

  it('starts reverse-flow history at the bottom without immediately loading older messages', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 400,
      scrollHeight: 1600,
      scrollTop: -600,
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

    await act(async () => {
      triggerHistoryIntersection();
      await Promise.resolve();
    });

    expect(scrollElement.scrollTop).toBe(0);
    expect(loadOlderHistory).not.toHaveBeenCalled();
  });

  it('loads older messages after the top sentinel intersects following initial placement', async () => {
    jest.useFakeTimers();
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
          hasOlderHistory: true,
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

    await act(async () => {
      triggerHistoryIntersection();
      await Promise.resolve();
    });
    expect(loadOlderHistory).not.toHaveBeenCalled();

    await act(async () => {
      triggerHistoryIntersection();
      await Promise.resolve();
    });

    expect(requireLatestHook(latestHook).olderHistoryLoadingPaused).toBe(true);
    expect(loadOlderHistory).not.toHaveBeenCalled();

    await act(async () => {
      jest.advanceTimersByTime(1499);
      await Promise.resolve();
    });

    expect(loadOlderHistory).not.toHaveBeenCalled();

    await act(async () => {
      jest.advanceTimersByTime(1);
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(loadOlderHistory).toHaveBeenCalledTimes(1);
  });

  it('does not compensate scrollTop when older rows are prepended in reverse flow', async () => {
    const scrollElement = createScrollElement({
      clientHeight: 300,
      scrollHeight: 900,
      scrollTop: -420,
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

    scrollElement.scrollTop = -420;
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

    expect(scrollElement.scrollTop).toBe(-420);
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
      scrollElement.scrollTop = -800;
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
      top: 0,
    });
    expect(scrollElement.scrollTop).toBe(0);
    expect(requireLatestHook(latestHook).showScrollToBottom).toBe(false);
  });

  function triggerHistoryIntersection() {
    if (!intersectionCallback) {
      throw new Error('IntersectionObserver callback was not registered');
    }

    intersectionCallback(
      [{ isIntersecting: true } as IntersectionObserverEntry],
      {} as IntersectionObserver,
    );
  }
});

interface HookProbeRenderState {
  olderHistoryLoadingPaused: boolean;
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
  hasOlderHistory: boolean;
  layoutSignature: string;
  loadingOlderHistory?: boolean;
  loadOlderHistory: () => Promise<void>;
  onRender?: (state: HookProbeRenderState) => void;
  rowCount: number;
  scrollElement: ScrollElement;
}) {
  const hook = useMessageListScroll({
    hasOlderHistory: props.hasOlderHistory,
    layoutSignature: props.layoutSignature,
    loadOlderHistory: props.loadOlderHistory,
    loadingOlderHistory: props.loadingOlderHistory ?? false,
    rowCount: props.rowCount,
  });
  props.onRender?.({
    olderHistoryLoadingPaused: hook.olderHistoryLoadingPaused,
    scrollToBottom: hook.scrollToBottom,
    showScrollToBottom: hook.showScrollToBottom,
  });

  return React.createElement(
    'div',
    { ref: hook.scrollElementRef, 'data-node': 'scroll' },
    React.createElement('div', { ref: hook.historySentinelRef, 'data-node': 'sentinel' }),
  );
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

function createNodeMock(scrollElement: ScrollElement) {
  return (element: React.ReactElement) => {
    if (element.props['data-node'] === 'scroll') {
      return scrollElement;
    }
    if (element.props['data-node'] === 'sentinel') {
      return {};
    }

    return null;
  };
}
