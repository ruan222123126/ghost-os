import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { AssistantChatMessage, EventChatMessage, ToolChatMessage, UserChatMessage } from '@/lib/types';
import type { MessageRowProps } from './types';
import { MessageRow } from './MessageRow';

type AssistantMarkdownMockProps = {
  content: string;
  enabled?: boolean;
  showCopyButton?: boolean;
};

const assistantMarkdownMock = jest.fn((_props: AssistantMarkdownMockProps) => null);

jest.mock('./AssistantMarkdownContent', () => ({
  AssistantMarkdownContent: (props: AssistantMarkdownMockProps) => assistantMarkdownMock(props),
}));

describe('components/message/MessageRow', () => {
  beforeEach(() => {
    assistantMarkdownMock.mockClear();
  });

  it('renders tool card, image gallery, and attachment list together', () => {
    const message: ToolChatMessage = {
      id: 'tool-1',
      kind: 'tool',
      content: 'Generated 1 image(s).',
      toolName: 'screen_action',
      toolStatus: 'success',
      images: [
        {
          id: 'img-1',
          path: '/tmp/generated-image.png',
          mimeType: 'image/png',
          width: 512,
          height: 512,
          bytes: 1024,
        },
      ],
      attachments: [
        {
          artifactId: 'artifact-1',
          name: 'generated-image.png',
          downloadUrl: '/api/sessions/session-1/artifacts/artifact-1',
          mimeType: 'image/png',
          bytes: 1024,
        },
      ],
    };

    const html = renderMessageRow({
      message,
      assistantMarkdownEnabled: true,
      toolCallCompactOutputEnabled: false,
      loading: false,
      isToolCardOpen: false,
      onAnswerQuestion: async () => undefined,
      onCancelQuestion: async () => undefined,
      onToggleToolCard: () => undefined,
    });

    expect(html).toContain('tool-card');
    expect(html).toContain('message-image-gallery');
    expect(html).toContain('/tmp/generated-image.png');
    expect(html).toContain('generated-image.png');
  });

  it('passes assistant markdown toggle to markdown renderer', () => {
    const message: AssistantChatMessage = {
      id: 'assistant-1',
      kind: 'assistant',
      content: '# title',
    };

    renderMessageRow({
      message,
      assistantMarkdownEnabled: false,
      toolCallCompactOutputEnabled: false,
      loading: false,
      onAnswerQuestion: async () => undefined,
      onCancelQuestion: async () => undefined,
    });

    expect(assistantMarkdownMock).toHaveBeenCalledWith(expect.objectContaining({
      content: '# title',
      enabled: false,
      showCopyButton: true,
    }));
  });

  it('hides assistant copy actions while the reply is still streaming', () => {
    const message: AssistantChatMessage = {
      id: 'assistant-streaming',
      kind: 'assistant',
      content: 'partial answer',
      inProgress: true,
    };

    const html = renderMessageRow({
      message,
      assistantMarkdownEnabled: true,
      toolCallCompactOutputEnabled: false,
      loading: false,
      onAnswerQuestion: async () => undefined,
      onCancelQuestion: async () => undefined,
    });

    expect(html).not.toContain('copy-button');
    expect(assistantMarkdownMock).toHaveBeenCalledWith(expect.objectContaining({
      content: 'partial answer',
      showCopyButton: false,
    }));
  });

  it('renders assistant content without the system output label or icon', () => {
    const message: AssistantChatMessage = {
      id: 'assistant-2',
      kind: 'assistant',
      content: 'plain answer',
    };

    const html = renderMessageRow({
      message,
      assistantMarkdownEnabled: true,
      toolCallCompactOutputEnabled: false,
      loading: false,
      onAnswerQuestion: async () => undefined,
      onCancelQuestion: async () => undefined,
    });

    expect(html).not.toContain('// SYSTEM_OUTPUT');
    expect(html).not.toContain('message-channel-icon');
  });

  it('renders user content with a neutral backplate instead of the black bubble shell', () => {
    const message: UserChatMessage = {
      id: 'user-1',
      kind: 'user',
      content: 'plain prompt',
    };

    const html = renderMessageRow({
      message,
      assistantMarkdownEnabled: true,
      toolCallCompactOutputEnabled: false,
      loading: false,
      onAnswerQuestion: async () => undefined,
      onCancelQuestion: async () => undefined,
    });

    expect(html).toContain('message-user-backplate');
    expect(html).toContain('message-content');
    expect(html).toContain('plain prompt');
    expect(html).not.toContain('message-bubble');
    expect(html).not.toContain('message-user-corner');
    expect(html).not.toContain('// USER_INPUT');
  });

  it('keeps the first three lines visible before expanding long user content', () => {
    const message: UserChatMessage = {
      id: 'user-long',
      kind: 'user',
      content: 'line one\nline two\nline three\nline four',
    };

    withMockWindow(buildMessageMeasurementWindow('20px'), () => {
      const renderer = renderMessageRowClient({
        message,
        assistantMarkdownEnabled: true,
        toolCallCompactOutputEnabled: false,
        loading: false,
        onAnswerQuestion: async () => undefined,
        onCancelQuestion: async () => undefined,
      }, { userContentScrollHeight: 96 });

      expect(renderer.root.findByProps({
        className: 'message-content message-user-content is-collapsed',
      }).children).toContain(message.content);
      const toggle = renderer.root.findByProps({ className: 'message-user-expand-toggle' });
      expect(toggle.children).toHaveLength(1);
      expect(toggle.findByProps({ className: 'message-user-expand-icon' })).toBeTruthy();

      act(() => {
        toggle.props.onClick();
      });

      expect(renderer.root.findByProps({
        className: 'message-content message-user-content',
      }).children).toContain(message.content);
      const collapseToggle = renderer.root.findByProps({ className: 'message-user-expand-toggle' });
      expect(collapseToggle.props['aria-expanded']).toBe(true);
      expect(collapseToggle.findByProps({
        className: 'message-user-expand-icon is-expanded',
      })).toBeTruthy();

      act(() => {
        collapseToggle.props.onClick();
      });

      expect(renderer.root.findByProps({
        className: 'message-content message-user-content is-collapsed',
      }).children).toContain(message.content);
      expect(renderer.root.findByProps({ className: 'message-user-expand-toggle' }).props['aria-expanded']).toBe(false);
    });
  });

  it('moves the assistant copy action inline when tools follow', () => {
    const message: AssistantChatMessage = {
      id: 'assistant-3',
      kind: 'assistant',
      content: 'needs more tools',
    };

    const html = renderMessageRow({
      message,
      assistantMarkdownEnabled: true,
      hasTrailingTool: true,
      toolCallCompactOutputEnabled: false,
      loading: false,
      onAnswerQuestion: async () => undefined,
      onCancelQuestion: async () => undefined,
    });

    expect(html).toContain('message-assistant-frame has-trailing-tool');
    expect(html).toContain('message-actions is-inline-with-body');
  });

  it('renders task run events as a divider row', () => {
    const message: EventChatMessage = {
      id: 'event-1',
      kind: 'event',
      content: '工作流进入循环：loop-node 第 1/3 轮',
    };

    const html = renderMessageRow({
      message,
      assistantMarkdownEnabled: true,
      toolCallCompactOutputEnabled: false,
      loading: false,
      onAnswerQuestion: async () => undefined,
      onCancelQuestion: async () => undefined,
    });

    expect(html).toContain('message-row is-event');
    expect(html).toContain('message-event-divider');
    expect(html).toContain('工作流进入循环');
  });
});

type MessageRowTestProps = MessageRowProps;

function renderMessageRow(props: MessageRowTestProps): string {
  const providerProps = {
    initialLocale: 'en-US',
  } as React.ComponentProps<typeof WebLocaleProvider>;

  return renderToStaticMarkup(
    React.createElement(
      WebLocaleProvider,
      providerProps,
      React.createElement(MessageRow, withMessageRowDefaults(props)),
    ),
  );
}

function renderMessageRowClient(
  props: MessageRowTestProps,
  options: {
    userContentScrollHeight: number;
  },
): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(MessageRow, withMessageRowDefaults(props)),
      {
        createNodeMock: (element) => createMessageRowNodeMock(element, options),
      },
    );
  });

  return renderer;
}

function withMessageRowDefaults(props: MessageRowTestProps): MessageRowProps {
  return props;
}

function createMessageRowNodeMock(
  element: React.ReactElement,
  options: {
    userContentScrollHeight: number;
  },
): Record<string, unknown> {
  if (classNameForElement(element).includes('message-user-content')) {
    return { scrollHeight: options.userContentScrollHeight };
  }
  return {};
}

function classNameForElement(element: React.ReactElement): string {
  const props = element.props as { className?: unknown };
  return typeof props.className === 'string' ? props.className : '';
}

function buildMessageMeasurementWindow(lineHeight: string) {
  return {
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    getComputedStyle: () => ({ lineHeight }),
  };
}

function withMockWindow<T>(windowValue: ReturnType<typeof buildMessageMeasurementWindow>, run: () => T): T {
  const globalObject = globalThis as { window?: unknown };
  const originalWindow = globalObject.window;
  globalObject.window = windowValue;
  try {
    return run();
  } finally {
    globalObject.window = originalWindow;
  }
}
