import type { FC } from 'react';
import { memo, useMemo } from 'react';
import { isMarkdownFenceLine, shouldRenderAssistantMarkdown } from './assistantMarkdown';
import { AssistantMarkdownRenderer } from './AssistantMarkdownRenderer';

interface AssistantMarkdownContentProps {
  content: string;
  enabled?: boolean;
  final?: boolean;
  showCopyButton?: boolean;
}

interface StreamingMarkdownBlock {
  content: string;
  key: string;
}

const AssistantMarkdownBlock = memo((props: {
  content: string;
  final: boolean;
  showCopyButton: boolean;
}) => (
  <AssistantMarkdownRenderer
    content={props.content}
    final={props.final}
    showCopyButton={props.showCopyButton}
  />
));
AssistantMarkdownBlock.displayName = 'AssistantMarkdownBlock';

const AssistantMarkdownContentBase: FC<AssistantMarkdownContentProps> = ({
  content,
  enabled = true,
  final = true,
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
      {final ? (
        <AssistantMarkdownBlock content={content} final showCopyButton={showCopyButton} />
      ) : (
        <StreamingAssistantMarkdownContent content={content} showCopyButton={showCopyButton} />
      )}
    </div>
  );
};

export const AssistantMarkdownContent = memo(AssistantMarkdownContentBase);
AssistantMarkdownContent.displayName = 'AssistantMarkdownContent';

const StreamingAssistantMarkdownContent: FC<{
  content: string;
  showCopyButton: boolean;
}> = ({ content, showCopyButton }) => {
  const blocks = useMemo(() => splitStreamingMarkdownBlocks(content), [content]);

  return (
    <>
      {blocks.stableBlocks.map((block) => (
        <AssistantMarkdownBlock
          key={block.key}
          content={block.content}
          final
          showCopyButton={showCopyButton}
        />
      ))}
      {blocks.activeBlock ? (
        <AssistantMarkdownBlock
          content={blocks.activeBlock}
          final={false}
          showCopyButton={showCopyButton}
        />
      ) : null}
    </>
  );
};

function splitStreamingMarkdownBlocks(content: string): {
  activeBlock: string;
  stableBlocks: StreamingMarkdownBlock[];
} {
  const stableBlocks: StreamingMarkdownBlock[] = [];
  let blockStart = 0;
  let cursor = 0;
  let inFence = false;

  for (const line of content.match(/[^\n]*(?:\n|$)/g) ?? []) {
    if (!line) {
      break;
    }

    const lineEnd = cursor + line.length;
    const lineText = line.endsWith('\n') ? line.slice(0, -1) : line;
    if (isMarkdownFenceLine(lineText)) {
      inFence = !inFence;
    }

    if (!inFence && lineText.trim() === '' && lineEnd < content.length) {
      const block = content.slice(blockStart, lineEnd);
      if (block.trim()) {
        stableBlocks.push({
          content: block,
          key: `markdown-block:${blockStart}`,
        });
      }
      blockStart = lineEnd;
    }

    cursor = lineEnd;
  }

  return {
    activeBlock: content.slice(blockStart),
    stableBlocks,
  };
}
