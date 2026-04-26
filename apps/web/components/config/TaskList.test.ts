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
});

function renderTaskList(
  props: React.ComponentProps<typeof TaskList>,
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
    return renderToStaticMarkup(
      React.createElement(
        WebLocaleProvider,
        {
          children: React.createElement(TaskList, props),
          initialLocale: locale,
        },
      ),
    );
  } finally {
    (globalThis as { window?: unknown }).window = originalWindow;
  }
}

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

function createAgentTask(input: { id: string; enabled: boolean; message: string }): AgentMessageTaskPayload {
  return {
    id: input.id,
    message: input.message,
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
