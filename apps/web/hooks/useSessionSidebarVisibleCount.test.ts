import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { useSessionSidebarVisibleCount } from './useSessionSidebarVisibleCount';

describe('hooks/useSessionSidebarVisibleCount', () => {
  it('shows 30 sessions initially and loads more near the bottom', async () => {
    const scrollElement = createScrollElement({
      scrollHeight: 1200,
      clientHeight: 500,
      scrollTop: 0,
    });

    let latestCount = 0;
    const ref = createScrollRef(scrollElement);

    await renderHook({
      scrollElementRef: ref,
      totalSessions: 80,
      resetKey: 'flat:',
      onRender: (count) => {
        latestCount = count;
        if (count > 0) {
          scrollElement.scrollHeight = count * 40;
        }
      },
    });

    expect(latestCount).toBe(30);

    await act(async () => {
      scrollElement.scrollTop = scrollElement.scrollHeight - scrollElement.clientHeight - 100;
      scrollElement.dispatchEvent(new Event('scroll'));
      await Promise.resolve();
    });

    expect(latestCount).toBe(60);
  });

  it('resets the visible count when the reset key changes', async () => {
    const scrollElement = createScrollElement({
      scrollHeight: 1200,
      clientHeight: 500,
      scrollTop: 1200 - 500 - 100,
    });
    const ref = createScrollRef(scrollElement);

    let latestCount = 0;
    let renderer!: TestRenderer.ReactTestRenderer;
    await act(async () => {
      renderer = TestRenderer.create(React.createElement(HookProbe, {
        scrollElementRef: ref,
        totalSessions: 80,
        resetKey: 'flat:',
        onRender: (count: number) => {
          latestCount = count;
          if (count > 0) {
            scrollElement.scrollHeight = count * 40;
          }
        },
      }));
      await Promise.resolve();
    });

    expect(latestCount).toBe(60);

    await act(async () => {
      scrollElement.scrollTop = 0;
      renderer.update(React.createElement(HookProbe, {
        scrollElementRef: ref,
        totalSessions: 80,
        resetKey: 'flat:search',
        onRender: (count: number) => {
          latestCount = count;
          if (count > 0) {
            scrollElement.scrollHeight = count * 40;
          }
        },
      }));
      await Promise.resolve();
    });

    expect(latestCount).toBe(30);
  });
});

function HookProbe(props: {
  scrollElementRef: React.RefObject<HTMLDivElement>;
  totalSessions: number;
  resetKey: string;
  onRender: (count: number) => void;
}) {
  const count = useSessionSidebarVisibleCount({
    scrollElementRef: props.scrollElementRef,
    totalSessions: props.totalSessions,
    resetKey: props.resetKey,
  });
  props.onRender(count);
  return null;
}

async function renderHook(props: {
  scrollElementRef: React.RefObject<HTMLDivElement>;
  totalSessions: number;
  resetKey: string;
  onRender: (count: number) => void;
}) {
  await act(async () => {
    TestRenderer.create(React.createElement(HookProbe, props));
    await Promise.resolve();
  });
}

function createScrollElement(layout: {
  scrollHeight: number;
  clientHeight: number;
  scrollTop: number;
}) {
  const listeners = new Map<string, Set<EventListener>>();
  return {
    scrollHeight: layout.scrollHeight,
    clientHeight: layout.clientHeight,
    scrollTop: layout.scrollTop,
    addEventListener: (type: string, listener: EventListener) => {
      const current = listeners.get(type) ?? new Set<EventListener>();
      current.add(listener);
      listeners.set(type, current);
    },
    removeEventListener: (type: string, listener: EventListener) => {
      listeners.get(type)?.delete(listener);
    },
    dispatchEvent: (event: Event) => {
      for (const listener of listeners.get(event.type) ?? []) {
        listener(event);
      }
      return true;
    },
  };
}

function createScrollRef(
  scrollElement: ReturnType<typeof createScrollElement>,
): React.RefObject<HTMLDivElement> {
  return {
    current: scrollElement as unknown as HTMLDivElement,
  };
}
