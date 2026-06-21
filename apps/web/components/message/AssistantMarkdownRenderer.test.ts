import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { AssistantMarkdownRenderer } from './AssistantMarkdownRenderer';

jest.mock('react-markdown', () => ({
  __esModule: true,
  default: ({
    children,
    components,
    rehypePlugins,
    remarkPlugins,
  }: {
    children: string;
    components?: Record<string, (props: Record<string, unknown>) => React.ReactElement>;
    rehypePlugins?: unknown[];
    remarkPlugins?: unknown[];
  }) => {
    const mathMatch = children.match(/\$\$([\s\S]*?)\$\$/);
    if (mathMatch) {
      return React.createElement(
        'div',
        {
          className: 'mock-markdown',
          'data-math-plugin-count': String(remarkPlugins?.length ?? 0),
          'data-rehype-plugin-count': String(rehypePlugins?.length ?? 0),
        },
        React.createElement('span', { className: 'katex' }, mathMatch[1]),
      );
    }

    const codeFenceMatch = children.match(/```([\w-]*)\n([\s\S]*?)```/);
    if (codeFenceMatch && components?.pre) {
      const languageClassName = codeFenceMatch[1] ? `language-${codeFenceMatch[1]}` : undefined;
      return React.createElement(
        'div',
        { className: 'mock-markdown' },
        components.pre({
          children: React.createElement('code', { className: languageClassName }, codeFenceMatch[2]),
        }),
      );
    }

    return React.createElement('div', { className: 'mock-markdown' }, children);
  },
}));

jest.mock('remark-gfm', () => ({
  __esModule: true,
  default: () => undefined,
}));

jest.mock('remark-math', () => ({
  __esModule: true,
  default: () => undefined,
}));

jest.mock('rehype-katex', () => ({
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

describe('components/message/AssistantMarkdownRenderer', () => {
  it('renders markdown through the renderer', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownRenderer, {
        content: '# Heading',
      }),
    );

    expect(html).toContain('mock-markdown');
  });

  it('enables math plugins for double-dollar formulas', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownRenderer, {
        content: '如果意思是 9：$$\\frac{6}{2}(1+2)$$',
      }),
    );

    expect(html).toContain('katex');
    expect(html).toContain('\\frac{6}{2}(1+2)');
    expect(html).toContain('data-math-plugin-count="2"');
    expect(html).toContain('data-rehype-plugin-count="1"');
  });

  it('renders copy button for fenced code blocks', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownRenderer, {
        content: '```ts\nconsole.log(1)\n```',
      }),
    );

    expect(html).toContain('assistant-code-block');
    expect(html).toContain('assistant-code-language');
    expect(html).toContain('TYPESCRIPT');
    expect(html).toContain('mock-copy-button');
    expect(html).toContain('data-copy-text=\"console.log(1)\"');
    expect(html).toContain('data-variant=\"code\"');
  });

  it('hides fenced code copy button while the reply is still streaming', () => {
    const html = renderToStaticMarkup(
      React.createElement(AssistantMarkdownRenderer, {
        content: '```ts\nconsole.log(1)\n```',
        showCopyButton: false,
      }),
    );

    expect(html).toContain('assistant-code-block');
    expect(html).not.toContain('mock-copy-button');
  });
});
