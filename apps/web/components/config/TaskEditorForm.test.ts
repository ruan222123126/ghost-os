import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { createTaskEditorState } from '@/lib/configTasks';
import { TaskEditorForm } from './TaskEditorForm';

describe('components/config/TaskEditorForm', () => {
  it('shows task type selector only when creating a task', () => {
    const createHTML = renderTaskEditorForm('create', createTaskEditorState(null));
    const editHTML = renderTaskEditorForm('edit', createTaskEditorState(null));

    expect(createHTML).toContain('Task Type');
    expect(createHTML).toContain('Text Task');
    expect(createHTML).toContain('Loop Task');
    expect(editHTML).not.toContain('Task Type');
  });

  it('renders loop fields from the unified task editor', () => {
    const editor = {
      ...createTaskEditorState(null),
      taskType: 'loop' as const,
      relayStopPolicy: 'ai_decides' as const,
    };

    const html = renderTaskEditorForm('create', editor);

    expect(html).toContain('Total Task');
    expect(html).toContain('Completion Rule');
    expect(html).toContain('AI decides completion');
    expect(html).toContain('Max Loop Rounds');
    expect(html).toContain('Preset');
    expect(html).toContain('Total Timeout');
    expect(html).not.toContain('Session ID');
    expect(html).not.toContain('Custom Runtime Config');
  });
});

function renderTaskEditorForm(
  mode: React.ComponentProps<typeof TaskEditorForm>['editorMode'],
  editor: React.ComponentProps<typeof TaskEditorForm>['editor'],
): string {
  return renderToStaticMarkup(
    React.createElement(
      WebLocaleProvider,
      { initialLocale: 'en-US' },
      React.createElement(TaskEditorForm, {
        editorMode: mode,
        editor,
        presets: [],
        sessionOptions: [],
        controlsDisabled: false,
        saving: false,
        onChangeEditor: () => undefined,
        onSelectTaskType: () => undefined,
        onSubmit: async () => undefined,
        onCancelEditing: () => undefined,
      }),
    ),
  );
}
