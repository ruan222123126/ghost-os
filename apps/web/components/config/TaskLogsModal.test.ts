import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { buildAssistantMessage } from '@/lib/chatMessages';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { TaskRunLog } from '@/lib/types';
import { TaskLogsModal } from './TaskLogsModal';

jest.mock('@/hooks/config/useLiveRunViewer', () => ({
  useLiveRunViewer: ({ run }: { run: TaskRunLog }) => {
    return {
      cards: [{
        card_id: 'card-1',
        kind: 'agent_task',
        title: 'run-live',
        status: 'running',
        started_at: '2026-05-29T00:00:02Z',
        source_events: [],
      }],
      followLatest: true,
      output: {
        committedMessages: [buildAssistantMessage('working', 'assistant-1')],
        streamingRows: [],
      },
      selectedCard: {
        card_id: 'card-1',
        kind: 'agent_task',
        title: 'run-live',
        status: 'running',
        started_at: '2026-05-29T00:00:02Z',
        source_events: [],
      },
      selectCard: jest.fn(),
      streamError: '',
      sourceSessionError: '',
    };
  },
}));

describe('components/config/TaskLogsModal', () => {
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
    expect(textContent(renderer.root)).toContain('AI Output');
    expect(textContent(renderer.root)).toContain('working');

    act(() => renderer.unmount());
  });

  it('disables live view when no session id is available', () => {
    const renderer = renderModal({
      logs: [buildRunLog({
        status: 'success',
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
