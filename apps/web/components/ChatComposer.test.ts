import React from 'react';
import TestRenderer, { act, type ReactTestInstance } from 'react-test-renderer';
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

  it('opens attachment menu from the plus button', () => {
    const harness = renderComposerHarness({ onSelectFiles: jest.fn() });

    act(() => {
      harness.plusButton().props.onClick();
    });

    expect(harness.menuItem('Photo')).toBeTruthy();
    expect(harness.menuItem('File')).toBeTruthy();
    expect(harness.menuItem('Skill')).toBeTruthy();
  });

  it('opens image picker from the photo menu item', () => {
    const fileInputClick = jest.fn();
    const harness = renderComposerHarness({
      fileInputClick,
      onSelectFiles: jest.fn(),
    });

    act(() => {
      harness.plusButton().props.onClick();
    });

    act(() => {
      harness.menuItem('Photo').props.onClick();
    });

    expect(fileInputClick).toHaveBeenCalledTimes(1);
    expect(harness.menuItems()).toEqual([]);
  });

  it('renders file and skill menu items as disabled placeholders', () => {
    const fileInputClick = jest.fn();
    const harness = renderComposerHarness({
      fileInputClick,
      onSelectFiles: jest.fn(),
    });

    act(() => {
      harness.plusButton().props.onClick();
    });

    expect(harness.menuItem('File').props.disabled).toBe(true);
    expect(harness.menuItem('Skill').props.disabled).toBe(true);
    expect(harness.menuItem('File').props.onClick).toBeUndefined();
    expect(harness.menuItem('Skill').props.onClick).toBeUndefined();
    expect(fileInputClick).not.toHaveBeenCalled();
  });

  it('closes the attachment menu after submitting', async () => {
    const harness = renderComposerHarness({ initialValue: 'send me', onSelectFiles: jest.fn() });

    act(() => {
      harness.plusButton().props.onClick();
    });

    expect(harness.menuItem('Photo')).toBeTruthy();

    await act(async () => {
      harness.form().props.onSubmit({
        preventDefault: jest.fn(),
      });
    });

    expect(harness.submissions).toEqual(['send me']);
    expect(harness.menuItems()).toEqual([]);
  });
});

function renderComposerHarness(options: {
  fileInputClick?: () => void;
  initialValue?: string;
  onSelectFiles?: (files: FileList) => Promise<void> | void;
} = {}) {
  const submissions: string[] = [];
  let renderer!: TestRenderer.ReactTestRenderer;
  const { fileInputClick, initialValue = '', onSelectFiles } = options;

  function Harness() {
    const [value, setValue] = React.useState(initialValue);
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
        onSelectFiles,
      }),
    );
  }

  act(() => {
    renderer = TestRenderer.create(React.createElement(Harness), {
      createNodeMock: (element) => {
        if (element.type !== 'input') {
          return null;
        }

        return {
          click: fileInputClick ?? jest.fn(),
        };
      },
    });
  });

  return {
    form: () => renderer.root.findByType('form'),
    menuItem: (label: string) => findButtonByText(renderer, label),
    menuItems: () => renderer.root.findAllByProps({ role: 'menuitem' }),
    plusButton: () => renderer.root.findByProps({ className: 'composer-plus-btn' }),
    submissions,
    textarea: () => renderer.root.findByType('textarea'),
  };
}

function findButtonByText(renderer: TestRenderer.ReactTestRenderer, text: string): ReactTestInstance {
  const button = renderer.root.findAllByType('button').find((node) => node.props.children === text);
  if (!button) {
    throw new Error(`Button not found: ${text}`);
  }

  return button;
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
