import React from 'react';
import TestRenderer, { act, type ReactTestInstance } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { AgentRuntimeType, ChatSelectedSkill, SkillPayload } from '@/lib/types';
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

    expect(harness.menuItem('File')).toBeTruthy();
    expect(harness.menuItem('Feature')).toBeTruthy();
    expect(harness.menuItem('Skill')).toBeTruthy();
  });

  it('opens file picker from the file menu item', () => {
    const fileInputClick = jest.fn();
    const harness = renderComposerHarness({
      fileInputClick,
      onSelectFiles: jest.fn(),
    });

    act(() => {
      harness.plusButton().props.onClick();
    });

    act(() => {
      harness.menuItem('File').props.onClick();
    });

    expect(fileInputClick).toHaveBeenCalledTimes(1);
    expect(harness.menuItems()).toEqual([]);
  });

  it('renders skill as disabled placeholder when no skill handler is wired', () => {
    const fileInputClick = jest.fn();
    const harness = renderComposerHarness({
      fileInputClick,
      onSelectFiles: jest.fn(),
    });

    act(() => {
      harness.plusButton().props.onClick();
    });

    expect(harness.menuItem('Skill').props.disabled).toBe(true);
    expect(harness.menuItem('Skill').props.onClick).toBeUndefined();
    expect(fileInputClick).not.toHaveBeenCalled();
  });

  it('opens the feature panel from the plus menu and toggles codex mode', () => {
    const harness = renderComposerHarness({ canEnableCodexMode: true });

    act(() => {
      harness.plusButton().props.onClick();
    });
    act(() => {
      harness.menuItem('Feature').props.onClick();
    });

    expect(harness.runtimeButton('Ghost').props['aria-pressed']).toBe(true);
    expect(harness.runtimeButton('Codex').props['aria-pressed']).toBe(false);

    act(() => {
      harness.runtimeButton('Codex').props.onClick();
    });

    act(() => {
      harness.plusButton().props.onClick();
    });
    act(() => {
      harness.menuItem('Feature').props.onClick();
    });

    expect(harness.runtimeButton('Ghost').props['aria-pressed']).toBe(false);
    expect(harness.runtimeButton('Codex').props['aria-pressed']).toBe(true);
  });

  it('disables codex mode in the feature menu when unavailable', () => {
    const harness = renderComposerHarness();

    act(() => {
      harness.plusButton().props.onClick();
    });
    act(() => {
      harness.menuItem('Feature').props.onClick();
    });

    expect(harness.runtimeButton('Codex').props.disabled).toBe(true);
    expect(harness.statusNode('Codex is unavailable right now')).toBeTruthy();
  });

  it('opens the skill panel from the plus menu and pins the selected skill', () => {
    const refreshSkills = jest.fn();
    const harness = renderComposerHarness({
      onRefreshSkills: refreshSkills,
      onSelectSkill: true,
      skills: [buildSkill({ id: 'skill_release', name: 'release_flow' })],
    });

    act(() => {
      harness.plusButton().props.onClick();
    });
    act(() => {
      harness.menuItem('Skill').props.onClick();
    });

    expect(refreshSkills).toHaveBeenCalledTimes(1);
    expect(harness.skillMenuItem('release_flow')).toBeTruthy();
    expect(harness.skillDescriptionNodes()).toEqual([]);

    act(() => {
      harness.skillMenuItem('release_flow').props.onClick();
    });

    expect(harness.selectedSkillButton().props.children).toBe('release_flow');
    expect(harness.textarea().props.placeholder).toBe('');
    expect(harness.menuItems()).toEqual([]);
  });

  it('clears the selected skill when backspace is pressed on an empty draft', () => {
    const harness = renderComposerHarness({
      initialSelectedSkill: { id: 'skill_release', name: 'release_flow' },
      onSelectSkill: true,
    });

    expect(harness.selectedSkillButton().props.children).toBe('release_flow');

    act(() => {
      harness.textarea().props.onKeyDown({
        key: 'Backspace',
        preventDefault: jest.fn(),
      });
    });

    expect(harness.selectedSkillButtons()).toEqual([]);
  });

  it('closes the attachment menu after submitting', async () => {
    const harness = renderComposerHarness({ initialValue: 'send me', onSelectFiles: jest.fn() });

    act(() => {
      harness.plusButton().props.onClick();
    });

    expect(harness.menuItem('File')).toBeTruthy();

    await act(async () => {
      harness.form().props.onSubmit({
        preventDefault: jest.fn(),
      });
    });

    expect(harness.submissions).toEqual(['send me']);
    expect(harness.menuItems()).toEqual([]);
  });

  it('focuses the textarea without scrolling as soon as submit is accepted', async () => {
    const pendingSubmit = createDeferred<void>();
    const textareaFocus = jest.fn();
    const onSubmit = jest.fn(async () => {
      await pendingSubmit.promise;
    });
    const harness = renderComposerHarness({
      initialValue: 'send me',
      onSubmit,
      textareaFocus,
    });

    await act(async () => {
      harness.form().props.onSubmit({
        preventDefault: jest.fn(),
      });
      await Promise.resolve();
    });

    expect(harness.submissions).toEqual(['send me']);
    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(textareaFocus).toHaveBeenCalledWith({ preventScroll: true });

    pendingSubmit.resolve();
    await act(async () => {
      await pendingSubmit.promise;
    });
  });

  it('keeps the composer expanded when the compact row would still wrap', async () => {
    const harness = renderComposerHarness({
      initialValue: 'line one\nline two',
      textareaLayout: {
        actualHeight: 40,
        compactHeight: 64,
        compactWidth: 240,
        multilineHeight: 64,
      },
    });

    expect(harness.composerRowClassName()).toContain('is-expanded');

    await act(async () => {
      harness.textarea().props.onChange({ target: { value: 'fits after expansion but wraps in compact layout' } });
    });

    expect(harness.composerRowClassName()).toContain('is-expanded');
  });

  it('collapses the composer once the compact row fits a single line', async () => {
    const harness = renderComposerHarness({
      initialValue: 'line one\nline two',
      textareaLayout: {
        actualHeight: 40,
        compactHeight: 40,
        compactWidth: 240,
        multilineHeight: 64,
      },
    });

    expect(harness.composerRowClassName()).toContain('is-expanded');

    await act(async () => {
      harness.textarea().props.onChange({ target: { value: 'short' } });
    });

    expect(harness.composerRowClassName()).not.toContain('is-expanded');
  });
});

interface ComposerTextareaLayoutMock {
  actualHeight: number;
  compactHeight: number;
  compactWidth: number;
  multilineHeight: number;
}

function renderComposerHarness(options: {
  canEnableCodexMode?: boolean;
  fileInputClick?: () => void;
  initialAgentRuntime?: AgentRuntimeType;
  initialValue?: string;
  initialSelectedSkill?: ChatSelectedSkill;
  onSubmit?: (value: string) => Promise<void> | void;
  onRefreshSkills?: () => Promise<void> | void;
  onSelectFiles?: (files: FileList) => Promise<void> | void;
  onSelectSkill?: boolean;
  skills?: SkillPayload[];
  textareaFocus?: (options?: FocusOptions) => void;
  textareaLayout?: ComposerTextareaLayoutMock;
} = {}) {
  const submissions: string[] = [];
  let renderer!: TestRenderer.ReactTestRenderer;
  const {
    canEnableCodexMode = false,
    fileInputClick,
    initialAgentRuntime = 'ghost',
    initialSelectedSkill = null,
    initialValue = '',
    onRefreshSkills,
    onSelectFiles,
    onSelectSkill = false,
    skills = [],
  } = options;
  let renderedValue = initialValue;
  const textareaStyle = { height: '', width: '' };
  const textareaLayout = options.textareaLayout ? createTextareaLayoutMock(options.textareaLayout) : null;

  function Harness() {
    const [agentRuntime, setAgentRuntime] = React.useState<AgentRuntimeType>(initialAgentRuntime);
    const [value, setValue] = React.useState(initialValue);
    const [selectedSkill, setSelectedSkill] = React.useState<ChatSelectedSkill | null>(initialSelectedSkill);
    renderedValue = value;
    return React.createElement(
      WebLocaleProvider,
      { initialLocale: 'en-US' },
      React.createElement(ChatComposer, {
        agentRuntime,
        canEnableCodexMode,
        value,
        onChange: setValue,
        onSubmit: async () => {
          submissions.push(value);
          await options.onSubmit?.(value);
        },
        sending: false,
        canSubmit: value.trim().length > 0 || selectedSkill !== null,
        selectedSkill,
        skills,
        onClearSelectedSkill: () => setSelectedSkill(null),
        onRefreshSkills,
        onSelectFiles,
        onSwitchAgentRuntime: setAgentRuntime,
        onSelectSkill: onSelectSkill
          ? (skill: SkillPayload) => setSelectedSkill({ id: skill.id, name: skill.name })
          : undefined,
      }),
    );
  }

  act(() => {
    renderer = TestRenderer.create(React.createElement(Harness), {
      createNodeMock: (element) => {
        if (element.type === 'textarea') {
          return {
            closest: textareaLayout ? () => textareaLayout.row : undefined,
            focus: options.textareaFocus ?? jest.fn(),
            get scrollHeight() {
              return getTextareaScrollHeight(textareaStyle, renderedValue, options.textareaLayout);
            },
            get value() {
              return renderedValue;
            },
            style: textareaStyle,
          };
        }

        if (element.type === 'input') {
          return {
            click: fileInputClick ?? jest.fn(),
          };
        }

        return null;
      },
    });
  });

  return {
    form: () => renderer.root.findByType('form'),
    composerRowClassName: () => findComposerRow(renderer).props.className as string,
    menuItem: (label: string) => findButtonContainingText(renderer, label),
    menuItems: () => renderer.root.findAllByProps({ role: 'menuitem' }),
    plusButton: () => renderer.root.findByProps({ className: 'composer-plus-btn' }),
    runtimeButton: (label: string) => findButtonContainingText(renderer, label),
    selectedSkillButton: () => renderer.root.findByProps({ className: 'composer-selected-skill' }),
    selectedSkillButtons: () => renderer.root.findAllByProps({ className: 'composer-selected-skill' }),
    skillDescriptionNodes: () => renderer.root.findAllByProps({ className: 'composer-skill-item-description' }),
    skillMenuItem: (label: string) => findButtonContainingText(renderer, label),
    statusNode: (text: string) => findNodeContainingText(renderer, text),
    submissions,
    textarea: () => renderer.root.findByType('textarea'),
  };
}

function createTextareaLayoutMock(layout: ComposerTextareaLayoutMock) {
  const attachmentWidth = 40;
  const actionsWidth = 72;
  const columnGap = 4;
  const rowWidth = layout.compactWidth + attachmentWidth + actionsWidth + columnGap * 2;
  const attachment = createElementWidthMock(attachmentWidth);
  const actions = createElementWidthMock(actionsWidth);

  return {
    row: {
      ...createElementWidthMock(rowWidth),
      querySelector: (selector: string) => {
        if (selector === '.composer-attachment-anchor') {
          return attachment;
        }
        if (selector === '.composer-actions') {
          return actions;
        }
        return null;
      },
    },
  };
}

function createElementWidthMock(width: number) {
  return {
    clientWidth: width,
    offsetWidth: width,
    getBoundingClientRect: () => ({ width }),
  };
}

function getTextareaScrollHeight(
  textareaStyle: { width: string },
  value: string,
  layout?: ComposerTextareaLayoutMock,
) {
  if (!layout) {
    return 0;
  }
  if (value.includes('\n')) {
    return layout.multilineHeight;
  }
  return textareaStyle.width === `${layout.compactWidth}px`
    ? layout.compactHeight
    : layout.actualHeight;
}

function buildSkill(overrides: Partial<SkillPayload> = {}): SkillPayload {
  return {
    id: 'skill_default',
    name: 'default_skill',
    description: 'Default skill',
    path: '/tmp/skill',
    source: 'repo',
    enabled: true,
    ...overrides,
  };
}

function findComposerRow(renderer: TestRenderer.ReactTestRenderer): ReactTestInstance {
  const row = renderer.root
    .findAll((node) => typeof node.props.className === 'string' && node.props.className.startsWith('composer-row'))
    .at(0);
  if (!row) {
    throw new Error('Composer row not found');
  }
  return row;
}

function findButtonContainingText(renderer: TestRenderer.ReactTestRenderer, text: string): ReactTestInstance {
  const button = renderer.root.findAllByType('button').find((node) => flattenChildrenText(node.props.children).includes(text));
  if (!button) {
    throw new Error(`Button not found: ${text}`);
  }

  return button;
}

function findNodeContainingText(renderer: TestRenderer.ReactTestRenderer, text: string): ReactTestInstance {
  const node = renderer.root.findAll((current) => flattenChildrenText(current.props.children).includes(text)).at(0);
  if (!node) {
    throw new Error(`Node not found: ${text}`);
  }

  return node;
}

function flattenChildrenText(children: unknown): string {
  if (typeof children === 'string') {
    return children;
  }
  if (Array.isArray(children)) {
    return children.map(flattenChildrenText).join('');
  }
  if (React.isValidElement(children)) {
    const element = children as React.ReactElement<{ children?: unknown }>;
    return flattenChildrenText(element.props.children);
  }
  return '';
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

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((promiseResolve) => {
    resolve = promiseResolve;
  });
  return { promise, resolve };
}
