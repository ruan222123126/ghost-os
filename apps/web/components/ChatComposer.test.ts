import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { ChatComposer } from './ChatComposer';

describe('components/ChatComposer', () => {
  it('does not submit enter while IME composition is active', async () => {
    const harness = renderComposerHarness();
    const textarea = harness.textarea();

    act(() => {
      textarea.props.onCompositionStart();
      textarea.props.onChange({ target: { value: '你好' } });
    });

    await act(async () => {
      textarea.props.onKeyDown(buildEnterEvent({
        isComposing: false,
        keyCode: 229,
      }));
    });

    expect(harness.submissions).toEqual([]);

    act(() => {
      textarea.props.onCompositionEnd();
    });

    await act(async () => {
      textarea.props.onKeyDown(buildEnterEvent());
    });

    expect(harness.submissions).toEqual(['你好']);
  });

  it('does not submit enter when browsers only expose IME keyCode 229', async () => {
    const harness = renderComposerHarness();
    const textarea = harness.textarea();

    act(() => {
      textarea.props.onChange({ target: { value: '继续' } });
    });

    await act(async () => {
      textarea.props.onKeyDown(buildEnterEvent({ keyCode: 229 }));
    });

    expect(harness.submissions).toEqual([]);
  });
});

function renderComposerHarness() {
  const submissions: string[] = [];
  let renderer!: TestRenderer.ReactTestRenderer;

  function Harness() {
    const [value, setValue] = React.useState('');
    return React.createElement(
      WebLocaleProvider,
      { initialLocale: 'en-US' },
      React.createElement(ChatComposer, {
        value,
        onChange: setValue,
        onSubmit: async () => {
          submissions.push(value);
        },
        sending: false,
      }),
    );
  }

  act(() => {
    renderer = TestRenderer.create(React.createElement(Harness));
  });

  return {
    submissions,
    textarea: () => renderer.root.findByType('textarea'),
  };
}

function buildEnterEvent(overrides: Partial<{
  isComposing: boolean;
  keyCode: number;
  which: number;
}> = {}) {
  return {
    key: 'Enter',
    shiftKey: false,
    ctrlKey: false,
    metaKey: false,
    altKey: false,
    preventDefault: jest.fn(),
    nativeEvent: {
      isComposing: overrides.isComposing ?? false,
      keyCode: overrides.keyCode ?? 13,
      which: overrides.which ?? overrides.keyCode ?? 13,
    },
  };
}
