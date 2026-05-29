import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { SessionDetail, TaskRunLog } from '@/lib/types';
import { TaskLogsModal } from './TaskLogsModal';

const getSession = jest.fn();
const streamSessionEvents = jest.fn();

jest.mock('@/hooks/config/useLiveRunViewer', () => ({
  useLiveRunViewer: ({ sessionId }: { sessionId: string }) => {
    streamSessionEvents({ sessionId });
    getSession(sessionId, { limit: 50 });
    return {
      events: [],
      session: buildSessionDetail(),
      streamError: '',
      snapshotError: '',
    };
  },
}));

jest.mock('@/lib/api/sessions/api', () => ({
  getSession: (...args: unknown[]) => getSession(...args),
}));

jest.mock('@/lib/api/sessions/events', () => ({
  streamSessionEvents: (...args: unknown[]) => streamSessionEvents(...args),
}));

describe('components/config/TaskLogsModal', () => {
  beforeEach(() => {
    getSession.mockReset();
    streamSessionEvents.mockReset();
    getSession.mockResolvedValue(buildSessionDetail());
    streamSessionEvents.mockResolvedValue(undefined);
  });

  it('opens the live run viewer for awaiting human logs with a session id', async () => {
    const renderer = renderModal({
      logs: [buildRunLog({
        status: 'awaiting_human',
        session_id_output: 'session-live',
      })],
    });

    const liveButton = findButtonByText(renderer.root, 'Live View');
    expect(liveButton.props.disabled).toBe(false);

    await act(async () => {
      liveButton.props.onClick(buildClickEvent());
      await Promise.resolve();
    });

    expect(textContent(renderer.root)).toContain('Live Run: run-live');
    expect(textContent(renderer.root)).toContain('session_id: session-live');
    expect(streamSessionEvents).toHaveBeenCalledWith(expect.objectContaining({
      sessionId: 'session-live',
    }));
    expect(getSession).toHaveBeenCalledWith('session-live', { limit: 50 });

    act(() => renderer.unmount());
  });

  it('disables live view for running logs without a session id', () => {
    const renderer = renderModal({
      logs: [buildRunLog({
        status: 'running',
        session_id_output: '',
      })],
    });

    const liveButton = findButtonByText(renderer.root, 'Live View');
    expect(liveButton.props.disabled).toBe(true);
    expect(findTitle(renderer.root, 'No session is available for live view yet.')).toBeTruthy();

    act(() => renderer.unmount());
  });
});

function renderModal(props: Partial<React.ComponentProps<typeof TaskLogsModal>>) {
  let renderer!: TestRenderer.ReactTestRenderer;
  const modalProps: React.ComponentProps<typeof TaskLogsModal> = {
    taskID: 'task-live',
    logs: [],
    loading: false,
    error: '',
    onClose: () => undefined,
    ...props,
  };

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(
        WebLocaleProvider,
        { initialLocale: 'en-US' },
        React.createElement(TaskLogsModal, modalProps),
      ),
    );
  });

  return renderer;
}

function buildRunLog(overrides: Partial<TaskRunLog>): TaskRunLog {
  return {
    task_id: 'task-live',
    run_id: 'run-live',
    trace_id: 'trace-live',
    scheduled_at: '2026-05-29T00:00:00Z',
    started_at: '2026-05-29T00:00:01Z',
    status: 'success',
    node_results: [],
    ...overrides,
  };
}

function buildSessionDetail(): SessionDetail {
  return {
    id: 'session-live',
    title: 'Live session',
    messages: [
      { index: 0, role: 'user', text: 'start' },
      { index: 1, role: 'assistant', text: 'working' },
    ],
    created_at: '2026-05-29T00:00:00Z',
    updated_at: '2026-05-29T00:00:02Z',
    message_count: 2,
    token_count: 12,
    page: {
      limit: 50,
      before: null,
      start_index: 0,
      end_index: 1,
      has_more_before: false,
      next_before: null,
    },
    turn_draft: {
      trace_id: 'trace-live',
      turn: 1,
      assistant_segments: [{ id: 'assistant-1', content: 'working' }],
      thinking_segments: [],
      tools: [],
      item_order: ['assistant-1'],
    },
  };
}

function buildClickEvent() {
  return {
    preventDefault: jest.fn(),
    stopPropagation: jest.fn(),
  };
}

function findButtonByText(
  root: TestRenderer.ReactTestInstance,
  label: string,
): TestRenderer.ReactTestInstance {
  const button = root.findAllByType('button').find((node) => textContent(node) === label);
  if (!button) {
    throw new Error(`Button not found: ${label}`);
  }
  return button;
}

function findTitle(root: TestRenderer.ReactTestInstance, title: string) {
  return root.find((node) => node.props.title === title);
}

function textContent(node: TestRenderer.ReactTestInstance): string {
  return node.children.map((child: string | number | TestRenderer.ReactTestInstance) => {
    if (typeof child === 'string' || typeof child === 'number') {
      return String(child);
    }
    return textContent(child);
  }).join('');
}
