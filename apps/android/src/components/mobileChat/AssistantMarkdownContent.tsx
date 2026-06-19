import type { ComponentPropsWithoutRef, ReactNode } from "react";
import { Children, isValidElement, memo, useMemo } from "react";
import ReactMarkdown from "react-markdown";
import type { Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import { extractCodeLanguage, formatCodeLanguageLabel, highlightCodeBlockHTML } from "./assistantMarkdown";
import { MessageCopyButton } from "./MessageCopyButton";

interface AssistantMarkdownContentProps {
  content: string;
  showCopyButton?: boolean;
}

interface CodeBlockData {
  className?: string;
  content: string;
}

const MARKDOWN_PLUGINS = [remarkGfm];

const BASE_MARKDOWN_COMPONENTS = {
  a: ({ href, children, ...props }) => (
    <a {...props} href={href} target="_blank" rel="noopener noreferrer" className="assistant-markdown-link">
      {children}
    </a>
  ),
  code: ({ className, children, ...props }) => (
    <code {...props} className={className}>
      {children}
    </code>
  ),
} satisfies Components;

function AssistantMarkdownContentBase({ content, showCopyButton = true }: AssistantMarkdownContentProps) {
  const components = useMemo<Components>(
    () => ({
      ...BASE_MARKDOWN_COMPONENTS,
      pre: ({ children, ...props }) => {
        const block = extractCodeBlockData(children);
        if (!block) {
          return <pre {...props}>{children}</pre>;
        }

        const codeContent = trimSingleTrailingLineBreak(block.content);
        const language = formatCodeLanguageLabel(extractCodeLanguage(block.className));

        return (
          <div className="assistant-code-block">
            <div className="assistant-code-header">
              <span className="assistant-code-language">{language}</span>
              {showCopyButton ? <MessageCopyButton text={codeContent} variant="code" /> : null}
            </div>
            <pre>
              <code
                className={block.className}
                dangerouslySetInnerHTML={{ __html: highlightCodeBlockHTML(codeContent) }}
              />
            </pre>
          </div>
        );
      },
    }),
    [showCopyButton],
  );

  return (
    <div className="assistant-markdown">
      <ReactMarkdown remarkPlugins={MARKDOWN_PLUGINS} components={components}>
        {content}
      </ReactMarkdown>
    </div>
  );
}

function extractCodeBlockData(children: ReactNode): CodeBlockData | null {
  const [first] = Children.toArray(children);
  if (!isValidElement<ComponentPropsWithoutRef<"code">>(first)) {
    return null;
  }

  return {
    className: first.props.className,
    content: flattenNodeText(first.props.children),
  };
}

function flattenNodeText(node: ReactNode): string {
  if (typeof node === "string" || typeof node === "number") {
    return String(node);
  }
  if (Array.isArray(node)) {
    return node.map((item) => flattenNodeText(item)).join("");
  }
  return "";
}

function trimSingleTrailingLineBreak(content: string): string {
  return content.endsWith("\n") ? content.slice(0, -1) : content;
}

export const AssistantMarkdownContent = memo(AssistantMarkdownContentBase);
