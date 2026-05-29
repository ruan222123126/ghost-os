import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import type { SessionMetadata } from '@/lib/types';
import { useSessions } from './useSessions';

export interface HookState {
  sessions: SessionMetadata[];
  currentSessionId: string;
  loading: boolean;
  error: string;
  loadSessions: (options?: {
    silent?: boolean;
    commitMode?: 'always' | 'when-new-session';
  }) => Promise<void>;
  deleteSession: (id: string) => Promise<void>;
  createNewSession: () => void;
  setCurrentSessionId: (id: string) => void;
}

interface HookProbeProps {
  autoRefresh?: boolean;
  onRender: (state: HookState) => void;
}

interface BrowserEventTargetMock {
  addEventListener: jest.Mock<void, [string, EventListener]>;
  removeEventListener: jest.Mock<void, [string, EventListener]>;
  dispatchEvent: jest.Mock<boolean, [Event]>;
}

interface BrowserDocumentMock extends BrowserEventTargetMock {
  documentElement: { lang: string };
  visibilityState: 'visible' | 'hidden';
}

export async function renderHook(props: HookProbeProps) {
  let renderer: TestRenderer.ReactTestRenderer;
  await act(async () => {
    renderer = TestRenderer.create(React.createElement(HookProbe, props));
    await Promise.resolve();
    await Promise.resolve();
  });
  return renderer!;
}

export async function unmountRenderer(renderer: TestRenderer.ReactTestRenderer) {
  await act(async () => {
    renderer.unmount();
    await Promise.resolve();
  });
}

export function snapshotState(state: HookState): HookState {
  return {
    ...state,
    sessions: [...state.sessions],
  };
}

export function createSession(id: string, updatedAt = '2026-05-16T00:00:00Z'): SessionMetadata {
  return {
    id,
    title: '',
    created_at: '2026-05-16T00:00:00Z',
    updated_at: updatedAt,
    message_count: 1,
    token_count: 1,
  };
}

export function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((nextResolve) => {
    resolve = nextResolve;
  });
  return { promise, resolve };
}

export function requireState(state: HookState | null): HookState {
  if (!state) {
    throw new Error('hook state missing');
  }
  return state;
}

export async function advanceBy(milliseconds: number) {
  await act(async () => {
    jest.advanceTimersByTime(milliseconds);
    await Promise.resolve();
    await Promise.resolve();
  });
}

export async function flushAsync() {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}

export function installBrowserMocks() {
  const windowTarget = createEventTargetMock();
  const documentTarget = createEventTargetMock();
  const documentMock = {
    ...documentTarget,
    documentElement: { lang: 'en' },
    visibilityState: 'visible' as 'visible' | 'hidden',
  };
  const windowMock = {
    ...windowTarget,
  };

  (globalThis as { window?: unknown }).window = windowMock;
  (globalThis as { document?: unknown }).document = documentMock;
}

export function setDocumentVisibilityState(value: 'visible' | 'hidden') {
  const documentMock = globalThis.document as unknown as BrowserDocumentMock;
  documentMock.visibilityState = value;
}

export function dispatchWindowEvent(type: string) {
  const windowMock = globalThis.window as unknown as BrowserEventTargetMock;
  windowMock.dispatchEvent({ type } as Event);
}

export function dispatchDocumentEvent(type: string) {
  const documentMock = globalThis.document as unknown as BrowserEventTargetMock;
  documentMock.dispatchEvent({ type } as Event);
}

function HookProbe(props: HookProbeProps) {
  const state = useSessions({
    autoRefresh: props.autoRefresh,
  });
  props.onRender(state);
  return null;
}

function createEventTargetMock(): BrowserEventTargetMock {
  const listeners = new Map<string, Set<EventListener>>();
  return {
    addEventListener: jest.fn((type: string, listener: EventListener) => {
      const current = listeners.get(type) ?? new Set<EventListener>();
      current.add(listener);
      listeners.set(type, current);
    }),
    removeEventListener: jest.fn((type: string, listener: EventListener) => {
      listeners.get(type)?.delete(listener);
    }),
    dispatchEvent: jest.fn((event: Event) => {
      for (const listener of listeners.get(event.type) ?? []) {
        listener(event);
      }
      return true;
    }),
  };
}
