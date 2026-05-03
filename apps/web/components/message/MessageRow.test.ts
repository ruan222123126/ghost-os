import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { AssistantChatMessage, ToolChatMessage } from '@/lib/types';
import type { MessageRowProps } from './types';
import { MessageRow } from './MessageRow';

const assistantMarkdownMock = jest.fn((_props: { content: string; enabled?: boolean }) => null);

jest.mock('./AssistantMarkdownContent', () => ({
  AssistantMarkdownContent: (props: { content: string; enabled?: boolean }) => assistantMarkdownMock(props),
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
});

function renderMessageRow(props: MessageRowProps): string {
  return renderToStaticMarkup(
    React.createElement(
      WebLocaleProvider,
      {
        initialLocale: 'en-US',
        children: React.createElement(MessageRow, props),
      },
    ),
  );
}
