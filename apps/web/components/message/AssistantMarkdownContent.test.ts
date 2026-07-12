import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

jest.mock('./AssistantMarkdownRenderer', () => ({
  AssistantMarkdownRenderer: ({ content, final }: { content: string; final?: boolean }) => React.createElement(
    'div',
    { className: 'mock-markdown-renderer', 'data-final': String(final) },
    content,
  ),
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

  it('adds display spacing between adjacent sentence outputs on the plain text path', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownContent, {
        content: '先检查项目。测试通过。',
        enabled: false,
      }),
    );

    expect(html).toContain('先检查项目。\n\n测试通过。');
  });

  it('keeps markdown recognition behavior when markdown is enabled', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownContent, {
        content: '# Heading',
        enabled: true,
      }),
    );

    expect(html).toContain('assistant-markdown');
    expect(html).toContain('mock-markdown-renderer');
    expect(html).toContain('data-final="true"');
  });

  it('renders stable streaming markdown blocks separately from the active block', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownContent, {
        content: '# Heading\n\n```ts\nconsole.log(1)',
        enabled: true,
        final: false,
      }),
    );

    expect(html).toContain('mock-markdown-renderer');
    expect(html).toContain('data-final="true"');
    expect(html).toContain('data-final="false"');
    expect(html).toContain('# Heading');
    expect(html).toContain('```ts\nconsole.log(1)');
  });

  it('does not split streaming tilde fences on blank lines inside code', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownContent, {
        content: '# Heading\n\n~~~python\nimport random\n\nprint(random.random())',
        enabled: true,
        final: false,
      }),
    );

    expect(html).toContain('data-final="true"');
    expect(html).toContain('data-final="false"');
    expect(html).toContain('~~~python\nimport random\n\nprint(random.random())');
  });

  it('keeps raw HTML content on the plain text path', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownContent, {
        content: '<strong>Safe</strong>',
        enabled: true,
      }),
    );

    expect(html).not.toContain('assistant-markdown');
    expect(html).not.toContain('mock-markdown-renderer');
    expect(html).toContain('&lt;strong&gt;Safe&lt;/strong&gt;');
  });
});
