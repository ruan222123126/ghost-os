import React from 'react';
import TestRenderer, { act, type ReactTestInstance } from 'react-test-renderer';
import { DEFAULT_CODEX_MODEL } from '@/lib/codexModels';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { ChatSendInput, ProviderModelOption } from '@/lib/types';
import { ChatInput } from './ChatInput';

jest.mock('@/hooks/useComposerSkills', () => ({
  useComposerSkills: () => ({
    refreshSkills: jest.fn(),
    skillError: '',
    skills: [],
    skillsLoading: false,
  }),
}));

describe('components/ChatInput', () => {
  it('sends normal codex mode through the codex runtime without leaking the ghost model', async () => {
    const harness = renderChatInputHarness({
      activeModel: {
        model: 'gpt-ghost',
        providerName: 'ghost',
        providerType: 'openai',
      },
      canEnableCodexMode: true,
    });

    await selectRuntimeMode(harness, 'normal');
    await setDraftAndSubmit(harness, 'ship release');

    expect(harness.sentInputs).toHaveLength(1);
    expect(harness.sentInputs[0]).toEqual(expect.objectContaining({
      agentRuntime: 'codex',
      codexMode: 'default',
      images: [],
      message: 'ship release',
      model: DEFAULT_CODEX_MODEL,
    }));
    expect(harness.sentInputs[0].mode).toBeUndefined();
  });

  it('keeps plan mode on the codex runtime and sends a codex provider model only', async () => {
    const harness = renderChatInputHarness({
      activeModel: {
        model: ' gpt-5-codex ',
        providerName: 'codex',
        providerType: 'codex',
      },
      canEnableCodexMode: true,
    });

    await selectRuntimeMode(harness, 'plan');
    await setDraftAndSubmit(harness, 'draft a plan');

    expect(harness.sentInputs).toEqual([
      expect.objectContaining({
        agentRuntime: 'codex',
        codexMode: 'plan',
        images: [],
        message: 'draft a plan',
        model: 'gpt-5-codex',
      }),
    ]);
    expect(harness.sentInputs[0].mode).toBeUndefined();
  });
});

function renderChatInputHarness(options: {
  activeModel?: ProviderModelOption;
  canEnableCodexMode?: boolean;
} = {}) {
  const sentInputs: ChatSendInput[] = [];
  let renderedValue = '';
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(
        WebLocaleProvider,
        { initialLocale: 'en-US' },
        React.createElement(ChatInput, {
          activeModel: options.activeModel,
          canEnableCodexMode: options.canEnableCodexMode,
          disabled: false,
          loading: false,
          onSend: async (input) => {
            sentInputs.push(input);
          },
        }),
      ),
      {
        createNodeMock: (element) => {
          if (element.type !== 'textarea') {
            return null;
          }

          return {
            closest: () => null,
            focus: jest.fn(),
            scrollHeight: 40,
            get value() {
              return renderedValue;
            },
            style: { height: '', width: '' },
          };
        },
      },
    );
  });

  return {
    form: () => renderer.root.findByType('form'),
    menuItem: (label: string) => findButtonContainingText(renderer, label),
    plusButton: () => renderer.root.findByProps({ className: 'composer-plus-btn' }),
    runtimeButton: (label: string) => findButtonContainingText(renderer, label),
    sentInputs,
    setRenderedValue: (value: string) => {
      renderedValue = value;
    },
    textarea: () => renderer.root.findByType('textarea'),
  };
}

async function selectRuntimeMode(
  harness: ReturnType<typeof renderChatInputHarness>,
  mode: 'normal' | 'plan',
): Promise<void> {
  await act(async () => {
    harness.plusButton().props.onClick();
  });
  await act(async () => {
    harness.menuItem('Feature').props.onClick();
  });
  await act(async () => {
    harness.runtimeButton(mode).props.onClick();
  });
}

async function setDraftAndSubmit(
  harness: ReturnType<typeof renderChatInputHarness>,
  draft: string,
): Promise<void> {
  harness.setRenderedValue(draft);
  await act(async () => {
    harness.textarea().props.onChange({ target: { value: draft } });
  });
  await act(async () => {
    await harness.form().props.onSubmit({ preventDefault: jest.fn() });
  });
}

function findButtonContainingText(
  renderer: TestRenderer.ReactTestRenderer,
  text: string,
): ReactTestInstance {
  const buttons = renderer.root.findAllByType('button');
  const match = buttons.find((button) => nodeContainsText(button, text));
  if (!match) {
    throw new Error(`button containing ${text} not found`);
  }
  return match;
}

function nodeContainsText(node: ReactTestInstance | string | number, text: string): boolean {
  if (typeof node === 'string' || typeof node === 'number') {
    return String(node).includes(text);
  }

  return node.children.some((child) => nodeContainsText(child as ReactTestInstance | string | number, text));
}
