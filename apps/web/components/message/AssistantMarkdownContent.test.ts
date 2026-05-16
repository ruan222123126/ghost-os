import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

jest.mock('react-markdown', () => ({
  __esModule: true,
  default: ({
    children,
    components,
  }: {
    children: string;
    components?: Record<string, (props: Record<string, unknown>) => React.ReactElement>;
  }) => {
    const codeFenceMatch = children.match(/```([\w-]*)\n([\s\S]*?)```/);
    if (codeFenceMatch && components?.pre) {
      const languageClassName = codeFenceMatch[1] ? `language-${codeFenceMatch[1]}` : undefined;
      return React.createElement(
        'div',
        {
          className: 'mock-markdown',
        },
        components.pre({
          children: React.createElement('code', { className: languageClassName }, codeFenceMatch[2]),
        }),
      );
    }

    return React.createElement(
      'div',
      {
        className: 'mock-markdown',
      },
      children,
    );
  },
}));

jest.mock('remark-gfm', () => ({
  __esModule: true,
  default: () => undefined,
}));

jest.mock('./MessageCopyButton', () => ({
  MessageCopyButton: ({ text, variant }: { text: string; variant?: string }) =>
    React.createElement('button', {
      className: 'mock-copy-button',
      'data-copy-text': text,
      'data-variant': variant ?? 'default',
      type: 'button',
    }),
}));

import { AssistantMarkdownContent } from './AssistantMarkdownContent';

describe('components/message/AssistantMarkdownContent', () => {
  it('renders assistant text as plain text when markdown is disabled', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownContent, {
        content: '# Heading',
        enabled: false,
      }),
    );

    expect(html).toContain('# Heading');
    expect(html).not.toContain('assistant-markdown');
    expect(html).not.toContain('mock-markdown');
  });

  it('keeps markdown recognition behavior when markdown is enabled', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownContent, {
        content: '# Heading',
        enabled: true,
      }),
    );

    expect(html).toContain('assistant-markdown');
    expect(html).toContain('mock-markdown');
  });

  it('keeps raw HTML content on the plain text path', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownContent, {
        content: '<strong>Safe</strong>',
        enabled: true,
      }),
    );

    expect(html).not.toContain('assistant-markdown');
    expect(html).not.toContain('mock-markdown');
    expect(html).toContain('&lt;strong&gt;Safe&lt;/strong&gt;');
  });

  it('renders copy button for fenced code blocks', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownContent, {
        content: '```ts\nconsole.log(1)\n```',
        enabled: true,
      }),
    );

    expect(html).toContain('assistant-code-block');
    expect(html).toContain('assistant-code-language');
    expect(html).toContain('TYPESCRIPT');
    expect(html).toContain('mock-copy-button');
    expect(html).toContain('data-copy-text=\"console.log(1)\"');
    expect(html).toContain('data-variant=\"code\"');
  });
});
