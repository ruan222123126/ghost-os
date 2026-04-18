import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import type { WebLocale } from '@/lib/i18n/locale';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { AgentMessageTaskPayload } from '@/lib/types';
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
