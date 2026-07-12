import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

jest.mock(
  'markstream-react',
  () => ({
    __esModule: true,
    default: ({
      batchRendering,
      codeBlockProps,
      content,
      customId,
      deferNodesUntilVisible,
      final,
      htmlPolicy,
      maxLiveNodes,
      typewriter,
    }: {
      batchRendering?: boolean;
      codeBlockProps?: { showCopyButton?: boolean };
      content: string;
      customId?: string;
      deferNodesUntilVisible?: boolean;
      final?: boolean;
      htmlPolicy?: string;
      maxLiveNodes?: number;
      typewriter?: boolean;
    }) => React.createElement(
      'div',
      {
        className: 'mock-markstream',
        'data-batch-rendering': String(batchRendering),
        'data-custom-id': customId,
        'data-defer-nodes-until-visible': String(deferNodesUntilVisible),
        'data-final': String(final),
        'data-html-policy': htmlPolicy,
        'data-max-live-nodes': String(maxLiveNodes),
        'data-show-copy': String(codeBlockProps?.showCopyButton),
        'data-typewriter': String(typewriter),
      },
      content,
    ),
    setCustomComponents: jest.fn(),
  }),
  { virtual: true },
);

import { AssistantMarkdownRenderer } from './AssistantMarkdownRenderer';

describe('components/message/AssistantMarkdownRenderer', () => {
  it('renders markdown through Markstream', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownRenderer, {
        content: '# Heading',
      }),
    );

    expect(html).toContain('mock-markstream');
    expect(html).toContain('# Heading');
    expect(html).toContain('data-custom-id="ghost-os-assistant-markdown"');
    expect(html).toContain('data-batch-rendering="false"');
    expect(html).toContain('data-defer-nodes-until-visible="false"');
    expect(html).toContain('data-max-live-nodes="0"');
    expect(html).toContain('data-final="true"');
    expect(html).toContain('data-html-policy="safe"');
  });

  it('marks streaming content as non-final without restarting a typewriter animation', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownRenderer, {
        content: '```ts\nconsole.log(1)',
        final: false,
      }),
    );

    expect(html).toContain('data-final="false"');
    expect(html).toContain('data-typewriter="false"');
  });

  it('forwards code copy visibility to Markstream code blocks', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownRenderer, {
        content: '```ts\nconsole.log(1)\n```',
        showCopyButton: false,
      }),
    );

    expect(html).toContain('data-show-copy="false"');
  });
});
