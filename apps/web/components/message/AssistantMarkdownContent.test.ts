import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

jest.mock('next/dynamic', () => ({
  __esModule: true,
  default: () => {
    return ({ content, final }: { content: string; final?: boolean }) => React.createElement(
      'div',
      { className: 'mock-dynamic-markdown', 'data-final': String(final) },
      content,
    );
  },
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
    expect(html).toContain('mock-dynamic-markdown');
    expect(html).toContain('data-final="true"');
  });

  it('keeps raw HTML content on the plain text path', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownContent, {
        content: '<strong>Safe</strong>',
        enabled: true,
      }),
    );

    expect(html).not.toContain('assistant-markdown');
    expect(html).not.toContain('mock-dynamic-markdown');
    expect(html).toContain('&lt;strong&gt;Safe&lt;/strong&gt;');
  });
});
