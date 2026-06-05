import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { buildAssistantMessage } from '@/lib/chatMessages';
import { WEB_LOCALE_STORAGE_KEY, type WebLocale } from '@/lib/i18n/locale';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { TaskRunLog } from '@/lib/types';
import { TaskLogsModal } from './TaskLogsModal';

const useLiveRunViewer = jest.fn();

jest.mock('@/hooks/config/useLiveRunViewer', () => ({
  useLiveRunViewer: (options: { run: TaskRunLog }) => useLiveRunViewer(options),
}));

beforeEach(() => {
  useLiveRunViewer.mockReset();
  useLiveRunViewer.mockImplementation(({ run }: { run: TaskRunLog }) => {
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
        committedMessages: [buildAssistantMessage(run.response_preview ?? 'working', 'assistant-1')],
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
  });
});

describe('components/config/TaskLogsModal', () => {
  it('opens the live run viewer for awaiting human logs with a session id', async () => {
    const renderer = renderModal({
      logs: [buildRunLog({
        status: 'awaiting_human',
        session_id_output: 'session-live',
      })],
    });

    const liveButton = findButtonByText(renderer.root, 'Details');
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

  it('refreshes the run snapshot before reopening live view', async () => {
    const refresh = jest.fn(async () => [
      buildRunLog({
        run_id: 'run-live',
        session_id_output: 'session-live',
        response_preview: 'fresh-from-refresh',
      }),
    ]);
    const renderer = renderModal({
      logs: [buildRunLog({
        run_id: 'run-live',
        status: 'awaiting_human',
        session_id_output: 'session-live',
        response_preview: 'stale-preview',
      })],
      onRefreshLogs: refresh,
    });

    await act(async () => {
      findButtonByText(renderer.root, 'Details').props.onClick(buildClickEvent());
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(refresh).toHaveBeenCalledWith('task-live');
    expect(useLiveRunViewer).toHaveBeenLastCalledWith({
      run: expect.objectContaining({
        run_id: 'run-live',
        response_preview: 'fresh-from-refresh',
      }),
    });
    expect(textContent(renderer.root)).toContain('fresh-from-refresh');

    act(() => renderer.unmount());
  });

  it('disables live view when no session id is available', () => {
    const renderer = renderModal({
      logs: [buildRunLog({
        status: 'success',
        session_id_output: '',
      })],
    });

    const liveButton = findButtonByText(renderer.root, 'Details');
    expect(liveButton.props.disabled).toBe(true);
    expect(findTitle(renderer.root, 'No details are available yet.')).toBeTruthy();

    act(() => renderer.unmount());
  });

  it('keeps run log cards collapsed by default', () => {
    const renderer = renderModal({
      logs: [buildRunLog({
        response_preview: 'ready',
      })],
    });

    const logCard = renderer.root.findByType('details');
    expect(logCard.props.open).toBeUndefined();

    act(() => renderer.unmount());
  });

  it('renders preview on a new bold-labelled block with a trailing ellipsis', () => {
    const renderer = renderModal({
      logs: [buildRunLog({
        response_preview: 'first line\nsecond line',
      })],
    });

    const previewLabel = renderer.root
      .findAllByType('p')
      .find((node) => textContent(node) === 'preview:');
    expect(previewLabel?.props.className).toContain('font-semibold');
    expect(textContent(renderer.root)).toContain('first line\nsecond line……');

    act(() => renderer.unmount());
  });

  it('formats timestamps and localizes statuses in Chinese mode', () => {
    const renderer = renderModal({
      logs: [buildRunLog({
        started_at: '2026-05-31T02:33:38.611Z',
        status: 'incomplete',
      })],
    }, 'zh-CN');

    const content = textContent(renderer.root);
    expect(content).toContain('2026-05-31 02:33:38');
    expect(content).toContain('未完成');
    expect(content).not.toContain('2026-05-31T02:33:38.611Z');

    act(() => renderer.unmount());
  });

  it('does not render node result headings or cards in collapsed log summaries', () => {
    const renderer = renderModal({
      logs: [buildRunLog({
        node_results: [{
          completed_seq: 1,
          node_id: 'group-1780193855320-0',
          node_type: 'group',
          status: 'success',
          preview: 'node result preview',
        }],
      })],
    });

    const content = textContent(renderer.root);
    expect(content).not.toContain('Node Results');
    expect(content).not.toContain('group-1780193855320-0');
    expect(content).not.toContain('node result preview');

    act(() => renderer.unmount());
  });

  it('renders a stop button for running logs and delegates clicks', async () => {
    const stopRun = jest.fn(async () => undefined);
    const renderer = renderModal({
      logs: [buildRunLog({
        status: 'running',
        session_id_output: 'session-live',
      })],
      onStopRun: stopRun,
    });

    await act(async () => {
      findButtonByText(renderer.root, 'Stop').props.onClick(buildClickEvent());
      await Promise.resolve();
    });

    expect(stopRun).toHaveBeenCalledWith(expect.objectContaining({
      run_id: 'run-live',
      status: 'running',
    }));

    act(() => renderer.unmount());
  });

  it('shows the stopping label while the current run is being cancelled', () => {
    const renderer = renderModal({
      logs: [buildRunLog({
        status: 'running',
        session_id_output: 'session-live',
      })],
      onStopRun: async () => undefined,
      stoppingRunId: 'run-live',
    });

    const stopButton = findButtonByText(renderer.root, 'Stopping...');
    expect(stopButton.props.disabled).toBe(true);

    act(() => renderer.unmount());
  });
});

function renderModal(
  props: Partial<React.ComponentProps<typeof TaskLogsModal>>,
  locale: WebLocale = 'en-US',
) {
  let renderer!: TestRenderer.ReactTestRenderer;
  const modalProps: React.ComponentProps<typeof TaskLogsModal> = {
    taskID: 'task-live',
    logs: [],
    loading: false,
    error: '',
    onRefreshLogs: undefined,
    onStopRun: undefined,
    stoppingRunId: '',
    onClose: () => undefined,
    ...props,
  };

  withWindowLocale(locale, () => {
    act(() => {
      renderer = TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          { initialLocale: locale },
          React.createElement(TaskLogsModal, modalProps),
        ),
      );
    });
  });

  return renderer;
}

function withWindowLocale(locale: WebLocale, run: () => void) {
  const descriptor = Object.getOwnPropertyDescriptor(globalThis, 'window');
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: {
      localStorage: {
        getItem: (key: string) => key === WEB_LOCALE_STORAGE_KEY ? locale : null,
        setItem: jest.fn(),
      },
      navigator: {
        language: locale,
      },
    },
  });
  try {
    run();
  } finally {
    if (descriptor) {
      Object.defineProperty(globalThis, 'window', descriptor);
      return;
    }
    delete (globalThis as { window?: unknown }).window;
  }
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
