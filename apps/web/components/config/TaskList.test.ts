import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import type { WebLocale } from '@/lib/i18n/locale';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { AgentMessageTaskPayload, WorkflowTaskPayload } from '@/lib/types';
import { TaskList } from './TaskList';

describe('components/config/TaskList', () => {
  it('renders enabled tasks before disabled tasks', () => {
    const html = renderTaskList({
      tasks: [
        createAgentTask({ id: 'task-disabled-a', enabled: false, message: 'disabled-a' }),
        createAgentTask({ id: 'task-enabled-a', enabled: true, message: 'enabled-a' }),
        createAgentTask({ id: 'task-enabled-b', enabled: true, message: 'enabled-b' }),
        createAgentTask({ id: 'task-disabled-b', enabled: false, message: 'disabled-b' }),
      ],
      loading: false,
      controlsDisabled: false,
      onEditTextTask: () => {},
      onEditWorkflowTask: () => {},
      onSetEnabled: async () => {},
      onRunNow: async () => {},
      onDelete: async () => {},
    });

    const enabledAIndex = html.indexOf('task-enabled-a');
    const enabledBIndex = html.indexOf('task-enabled-b');
    const disabledAIndex = html.indexOf('task-disabled-a');
    const disabledBIndex = html.indexOf('task-disabled-b');

    expect(enabledAIndex).toBeGreaterThan(-1);
    expect(enabledBIndex).toBeGreaterThan(-1);
    expect(disabledAIndex).toBeGreaterThan(-1);
    expect(disabledBIndex).toBeGreaterThan(-1);
    expect(enabledAIndex).toBeLessThan(enabledBIndex);
    expect(enabledBIndex).toBeLessThan(disabledAIndex);
    expect(disabledAIndex).toBeLessThan(disabledBIndex);
  });

  it('renders workflow total steps instead of agent-only steps', () => {
    const html = renderTaskList({
      tasks: [createWorkflowTask()],
      loading: false,
      controlsDisabled: false,
      onEditTextTask: () => {},
      onEditWorkflowTask: () => {},
      onSetEnabled: async () => {},
      onRunNow: async () => {},
      onDelete: async () => {},
    });

    expect(html).toContain('Workflow with 3 steps');
  });

  it('renders loop tasks in the unified task list', () => {
    const html = renderTaskList({
      tasks: [
        createAgentTask({
          id: 'loop-task',
          enabled: true,
          message: 'loop-message',
          agent_mode: 'relay',
          relay: { stop_policy: 'ai_decides', max_rounds: 5, execution_timeout_ms: 0 },
          runtime_overrides: { preset_id: 'preset-1' },
        }),
        createAgentTask({ id: 'text-task', enabled: true, message: 'text-message', agent_mode: 'single' }),
        createWorkflowTask(),
      ],
      presets: [{ id: 'preset-1', name: 'Research', prompt_refs: {}, tool_allowlist: [] }],
      loading: false,
      controlsDisabled: false,
      onEditTextTask: () => {},
      onEditWorkflowTask: () => {},
      onSetEnabled: async () => {},
      onRunNow: async () => {},
      onDelete: async () => {},
    });

    expect(html).toContain('loop-task');
    expect(html).toContain('loop-message');
    expect(html).toContain('Loop');
    expect(html).toContain('Stop: AI may finish early; force stop at 5 rounds');
    expect(html).toContain('Preset: Research');
    expect(html).toContain('text-task');
    expect(html).toContain('text-message');
    expect(html).toContain('workflow-task-1');
  });

  it('does not show the empty state when only loop tasks are provided', () => {
    const html = renderTaskList({
      tasks: [createAgentTask({
        id: 'loop-task',
        enabled: true,
        message: 'loop-message',
        agent_mode: 'relay',
        relay: { stop_policy: 'max_rounds', max_rounds: 3, execution_timeout_ms: 0 },
      })],
      loading: false,
      controlsDisabled: false,
      onEditTextTask: () => {},
      onEditWorkflowTask: () => {},
      onSetEnabled: async () => {},
      onRunNow: async () => {},
      onDelete: async () => {},
    });

    expect(html).not.toContain('No tasks configured yet.');
    expect(html).toContain('loop-task');
    expect(html).toContain('Stop: max 3 rounds');
  });

  it('renders logs button to the left of run button', () => {
    const html = renderTaskList({
      tasks: [createAgentTask({ id: 'task-one', enabled: true, message: 'task-one-message' })],
      loading: false,
      controlsDisabled: false,
      onEditTextTask: () => {},
      onEditWorkflowTask: () => {},
      onSetEnabled: async () => {},
      onRunNow: async () => {},
      onDelete: async () => {},
    });

    const logsIndex = html.indexOf('>Logs<');
    const runIndex = html.indexOf('>Run<');
    expect(logsIndex).toBeGreaterThan(-1);
    expect(runIndex).toBeGreaterThan(-1);
    expect(logsIndex).toBeLessThan(runIndex);
  });

  it('keeps text task runtime details out of the card summary', () => {
    const html = renderTaskList({
      tasks: [
        createAgentTask({
          id: 'preset-task',
          enabled: true,
          message: 'preset-message',
          runtime_overrides: {
            preset_id: 'preset-1',
            tool_allowlist_only: true,
            tool_allowlist: ['script_exec'],
          },
        }),
      ],
      loading: false,
      controlsDisabled: false,
      onEditTextTask: () => {},
      onEditWorkflowTask: () => {},
      onSetEnabled: async () => {},
      onRunNow: async () => {},
      onDelete: async () => {},
    });

    expect(html).toContain('preset-task');
    expect(html).toContain('preset-message');
    expect(html).not.toContain('Runtime: preset=preset-1 | tools=script_exec');
    expect(html).not.toContain('Mode: single');
  });
});

function renderTaskList(
  props: TaskListTestProps,
  locale: WebLocale = 'en-US',
): string {
  const originalWindow = (globalThis as { window?: unknown }).window;
  const localStorageMock = buildLocalStorageMock(locale);
  (globalThis as { window?: unknown }).window = {
    localStorage: localStorageMock,
    navigator: {
      language: locale,
    },
  };

  try {
    const taskListProps: React.ComponentProps<typeof TaskList> = {
      presets: [],
      logsTaskID: '',
      logsData: [],
      logsLoading: false,
      logsError: '',
      onOpenLogs: async () => {},
      onRefreshLogs: async () => [],
      onStopRun: async () => {},
      stoppingRunId: '',
      onCloseLogs: () => {},
      ...props,
    };
    return renderToStaticMarkup(
      React.createElement(
        WebLocaleProvider,
        {
          initialLocale: locale,
        },
        React.createElement(TaskList, taskListProps),
      ),
    );
  } finally {
    (globalThis as { window?: unknown }).window = originalWindow;
  }
}

type TaskListTestProps = Omit<
  React.ComponentProps<typeof TaskList>,
  'presets' | 'logsTaskID' | 'logsData' | 'logsLoading' | 'logsError' | 'onOpenLogs' | 'onRefreshLogs' | 'onStopRun' | 'stoppingRunId' | 'onCloseLogs'
> & Partial<Pick<
  React.ComponentProps<typeof TaskList>,
  'presets' | 'logsTaskID' | 'logsData' | 'logsLoading' | 'logsError' | 'onOpenLogs' | 'onRefreshLogs' | 'onStopRun' | 'stoppingRunId' | 'onCloseLogs'
>>;

function buildLocalStorageMock(locale: WebLocale): Storage {
  return {
    getItem: (key: string) => (key === 'ghost.web.locale' ? locale : null),
    setItem: () => undefined,
    removeItem: () => undefined,
    clear: () => undefined,
    key: () => null,
    get length() {
      return 1;
    },
  };
}

function createAgentTask(input: {
  id: string;
  enabled: boolean;
  message: string;
  agent_mode?: AgentMessageTaskPayload['agent_mode'];
  relay?: AgentMessageTaskPayload['relay'];
  runtime_overrides?: AgentMessageTaskPayload['runtime_overrides'];
}): AgentMessageTaskPayload {
  return {
    id: input.id,
    message: input.message,
    agent_mode: input.agent_mode,
    relay: input.relay,
    runtime_overrides: input.runtime_overrides,
    task_kind: 'agent_message',
    schedule_type: 'interval',
    interval_seconds: 300,
    enabled: input.enabled,
    created_at: '2026-04-12T00:00:00Z',
    updated_at: '2026-04-12T00:00:00Z',
  };
}

function createWorkflowTask(): WorkflowTaskPayload {
  return {
    id: 'workflow-task-1',
    task_kind: 'workflow',
    schedule_type: 'interval',
    interval_seconds: 600,
    enabled: true,
    created_at: '2026-04-12T00:00:00Z',
    updated_at: '2026-04-12T00:00:00Z',
    workflow: {
      nodes: [
        { id: 'start-1', type: 'start' },
        { id: 'agent-1', type: 'agent', agent: { message: 'step-1' } },
        { id: 'tool-1', type: 'tool', tool: { tool_name: 'script_exec', arguments: { command: 'echo hello' } } },
        { id: 'if-1', type: 'if', if: { operator: 'contains', value: 'ok', true_node_id: 'end-1', false_node_id: 'end-1' } },
        { id: 'end-1', type: 'end' },
      ],
      edges: [
        { from_node_id: 'start-1', to_node_id: 'agent-1' },
        { from_node_id: 'agent-1', to_node_id: 'tool-1' },
        { from_node_id: 'tool-1', to_node_id: 'if-1' },
        { from_node_id: 'if-1', to_node_id: 'end-1' },
      ],
    },
  };
}
