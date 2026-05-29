import type { ComponentPropsWithoutRef, FC, ReactNode } from 'react';
import { Children, isValidElement } from 'react';
import ReactMarkdown from 'react-markdown';
import type { Components } from 'react-markdown';
import remarkGfm from 'remark-gfm';
import {
  extractCodeLanguage,
  formatCodeLanguageLabel,
} from './assistantMarkdown';
import { MessageCopyButton } from './MessageCopyButton';

interface AssistantMarkdownRendererProps {
  content: string;
}

interface CodeBlockData {
  className?: string;
  content: string;
}

const MARKDOWN_PLUGINS = [remarkGfm];

const MARKDOWN_COMPONENTS: Components = {
  a: ({ href, children, ...props }) => (
    <a
      {...props}
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="assistant-markdown-link"
    >
      {children}
    </a>
  ),
  code: ({ className, children, ...props }) => (
    <code {...props} className={className}>
      {children}
    </code>
  ),
  pre: ({ children, ...props }) => {
    const block = extractCodeBlockData(children);
    if (!block) {
      return <pre {...props}>{children}</pre>;
    }

    const language = extractCodeLanguage(block.className);
    const languageLabel = formatCodeLanguageLabel(language);
    const codeContent = trimSingleTrailingLineBreak(block.content);
    return (
      <div className="assistant-code-block">
        <div className="assistant-code-header">
          <span className="assistant-code-language">{languageLabel}</span>
          <MessageCopyButton text={codeContent} variant="code" />
        </div>
        <pre>
          <code className={block.className}>{codeContent}</code>
        </pre>
      </div>
    );
  },
};

export const AssistantMarkdownRenderer: FC<AssistantMarkdownRendererProps> = ({
  content,
}) => {
  return (
    <ReactMarkdown
      remarkPlugins={MARKDOWN_PLUGINS}
      components={MARKDOWN_COMPONENTS}
    >
      {content}
    </ReactMarkdown>
  );
};

function extractCodeBlockData(children: ReactNode): CodeBlockData | null {
  const [first] = Children.toArray(children);
  if (!isValidElement<ComponentPropsWithoutRef<'code'>>(first)) {
    return null;
  }

  return {
    className: first.props.className,
    content: flattenNodeText(first.props.children),
  };
}

function flattenNodeText(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') {
    return String(node);
  }
  if (Array.isArray(node)) {
    return node.map((item) => flattenNodeText(item)).join('');
  }
  return '';
}

function trimSingleTrailingLineBreak(content: string): string {
  if (content.endsWith('\n')) {
    return content.slice(0, -1);
  }
  return content;
}
