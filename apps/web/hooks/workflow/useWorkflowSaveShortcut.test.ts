import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { useWorkflowSaveShortcut } from './useWorkflowSaveShortcut';

type KeydownListener = (event: KeyboardEvent) => void;

const originalWindowDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'window');
const keydownListeners = new Set<KeydownListener>();

describe('hooks/workflow/useWorkflowSaveShortcut', () => {
  let renderer: TestRenderer.ReactTestRenderer | undefined;

  beforeEach(() => {
    keydownListeners.clear();
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: {
        addEventListener: jest.fn((type: string, listener: EventListenerOrEventListenerObject) => {
          if (type === 'keydown' && typeof listener === 'function') {
            keydownListeners.add(listener as KeydownListener);
          }
        }),
        removeEventListener: jest.fn((type: string, listener: EventListenerOrEventListenerObject) => {
          if (type === 'keydown' && typeof listener === 'function') {
            keydownListeners.delete(listener as KeydownListener);
          }
        }),
      },
    });
  });

  afterEach(() => {
    act(() => {
      renderer?.unmount();
    });
    renderer = undefined;
    restoreWindow();
  });

  it('runs the latest save handler for Cmd/Ctrl+S', () => {
    const firstSave = jest.fn();
    const nextSave = jest.fn();

    act(() => {
      renderer = TestRenderer.create(React.createElement(HookProbe, { onSave: firstSave }));
    });
    const ctrlEvent = createKeyDownEvent({ ctrlKey: true, key: 's' });
    act(() => {
      dispatchKeyDown(ctrlEvent);
    });

    act(() => {
      renderer!.update(React.createElement(HookProbe, { onSave: nextSave }));
    });
    const metaEvent = createKeyDownEvent({ key: 'S', metaKey: true });
    act(() => {
      dispatchKeyDown(metaEvent);
    });

    expect(firstSave).toHaveBeenCalledTimes(1);
    expect(nextSave).toHaveBeenCalledTimes(1);
    expect(ctrlEvent.preventDefault).toHaveBeenCalledTimes(1);
    expect(metaEvent.preventDefault).toHaveBeenCalledTimes(1);
  });

  it('ignores non-save key events', () => {
    const onSave = jest.fn();

    act(() => {
      renderer = TestRenderer.create(React.createElement(HookProbe, { onSave }));
    });
    act(() => {
      dispatchKeyDown(createKeyDownEvent({ key: 's' }));
      dispatchKeyDown(createKeyDownEvent({ ctrlKey: true, key: 'x' }));
    });

    expect(onSave).not.toHaveBeenCalled();
  });
});

interface HookProbeProps {
  onSave: () => void;
}

function HookProbe(props: HookProbeProps) {
  useWorkflowSaveShortcut(props.onSave);
  return null;
}

interface TestKeyboardEvent {
  ctrlKey: boolean;
  key: string;
  metaKey: boolean;
  preventDefault: jest.Mock;
}

function createKeyDownEvent(options: { ctrlKey?: boolean; key: string; metaKey?: boolean }): TestKeyboardEvent {
  return {
    ctrlKey: options.ctrlKey ?? false,
    key: options.key,
    metaKey: options.metaKey ?? false,
    preventDefault: jest.fn(),
  };
}

function dispatchKeyDown(event: TestKeyboardEvent): void {
  for (const listener of keydownListeners) {
    listener(event as unknown as KeyboardEvent);
  }
}

function restoreWindow(): void {
  if (originalWindowDescriptor) {
    Object.defineProperty(globalThis, 'window', originalWindowDescriptor);
    return;
  }
  delete (globalThis as { window?: unknown }).window;
}
