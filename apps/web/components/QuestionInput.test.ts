import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { QuestionInput } from './QuestionInput';

describe('components/QuestionInput', () => {
  it('submits a single selected option through the answer callback', async () => {
    const onAnswer = jest.fn().mockResolvedValue(undefined);
    const renderer = renderQuestionInput({ onAnswer });

    await act(async () => {
      renderer.root.findAllByProps({ className: 'plan-question-option' })[0]?.props.onClick();
    });

    expect(onAnswer).toHaveBeenCalledWith('Use the recommended plan');
  });

  it('cancels the question when the skip button is selected', async () => {
    const onCancel = jest.fn().mockResolvedValue(undefined);
    const renderer = renderQuestionInput({ onCancel });

    await act(async () => {
      renderer.root.findByProps({ className: 'plan-question-skip' }).props.onClick();
    });

    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});

function renderQuestionInput(overrides: Partial<React.ComponentProps<typeof QuestionInput>> = {}) {
  let renderer: TestRenderer.ReactTestRenderer;
  act(() => {
    renderer = TestRenderer.create(
      React.createElement(
        WebLocaleProvider,
        { initialLocale: 'en-US' },
        React.createElement(QuestionInput, {
          loading: false,
          prompt: 'Which implementation should be used?',
          options: [
            { label: 'Use the recommended plan' },
            { label: 'Provide a different approach', allow_custom: true },
          ],
          onAnswer: async () => undefined,
          onCancel: async () => undefined,
          ...overrides,
        }),
      ),
    );
  });

  return renderer!;
}
