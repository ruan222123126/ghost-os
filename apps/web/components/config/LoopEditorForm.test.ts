import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { createLoopEditorState } from '@/lib/configLoops';
import { LoopEditorForm } from './LoopEditorForm';

describe('components/config/LoopEditorForm', () => {
  it('shows max rounds when AI decides completion', () => {
    const html = renderLoopEditorForm('ai_decides');

    expect(html).toContain('AI decides completion');
    expect(html).toContain('Max Loop Rounds');
  });

  it('shows max rounds when user max rounds is selected', () => {
    const html = renderLoopEditorForm('max_rounds');

    expect(html).toContain('Max Loop Rounds');
  });
});

function renderLoopEditorForm(stopPolicy: 'ai_decides' | 'max_rounds'): string {
  const editor = {
    ...createLoopEditorState(null),
    stopPolicy,
  };

  return renderToStaticMarkup(
    React.createElement(
      WebLocaleProvider,
      { initialLocale: 'en-US' },
      React.createElement(LoopEditorForm, {
        mode: 'create',
        editor,
        presets: [],
        controlsDisabled: false,
        saving: false,
        onChangeEditor: () => undefined,
        onSubmit: async () => true,
        onCancel: () => undefined,
      }),
    ),
  );
}
