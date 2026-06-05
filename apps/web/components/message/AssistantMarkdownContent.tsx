import nextDynamic from 'next/dynamic';
import type { FC } from 'react';
import { memo, useMemo } from 'react';
import { shouldRenderAssistantMarkdown } from './assistantMarkdown';

interface AssistantMarkdownContentProps {
  content: string;
  enabled?: boolean;
  showCopyButton?: boolean;
}

const AssistantMarkdownRenderer = nextDynamic(
  () => import('./AssistantMarkdownRenderer').then((mod) => mod.AssistantMarkdownRenderer),
  { ssr: false },
);

const AssistantMarkdownContentBase: FC<AssistantMarkdownContentProps> = ({
  content,
  enabled = true,
  showCopyButton = true,
}) => {
  const renderMode = useMemo(() => {
    if (!enabled) {
      return false;
    }
    return shouldRenderAssistantMarkdown(content);
  }, [content, enabled]);

  if (!renderMode) {
    return <div className="message-content">{content}</div>;
  }

  return (
    <div className="message-content assistant-markdown">
      <AssistantMarkdownRenderer content={content} showCopyButton={showCopyButton} />
    </div>
  );
};

export const AssistantMarkdownContent = memo(AssistantMarkdownContentBase);
AssistantMarkdownContent.displayName = 'AssistantMarkdownContent';
