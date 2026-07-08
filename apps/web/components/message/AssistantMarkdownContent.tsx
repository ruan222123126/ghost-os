import type { FC } from 'react';
import { memo, useMemo } from 'react';
import { formatAssistantTextForDisplay } from '../../../shared/assistantTextSpacing';
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
  active?: boolean;
  content: string;
  final: boolean;
  showCopyButton: boolean;
}) => {
  const className = props.active
    ? 'assistant-markdown-block is-active-streaming-block'
    : 'assistant-markdown-block';
  return (
    <div className={className}>
      <AssistantMarkdownRenderer
        content={props.content}
        final={props.final}
        showCopyButton={props.showCopyButton}
      />
    </div>
  );
});
AssistantMarkdownBlock.displayName = 'AssistantMarkdownBlock';

const AssistantMarkdownContentBase: FC<AssistantMarkdownContentProps> = ({
  content,
  enabled = true,
  final = true,
  showCopyButton = true,
}) => {
  const displayContent = useMemo(() => formatAssistantTextForDisplay(content), [content]);
  const renderMode = useMemo(() => {
    if (!enabled) {
      return false;
    }
    return shouldRenderAssistantMarkdown(displayContent);
  }, [displayContent, enabled]);

  if (!renderMode) {
    return <div className="message-content">{displayContent}</div>;
  }

  return (
    <div className="message-content assistant-markdown">
      {final ? (
        <AssistantMarkdownBlock content={displayContent} final showCopyButton={showCopyButton} />
      ) : (
        <StreamingAssistantMarkdownContent content={displayContent} showCopyButton={showCopyButton} />
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
          active
          content={completeUnclosedFenceBlock(blocks.activeBlock)}
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

function completeUnclosedFenceBlock(content: string): string {
  const fence = findUnclosedFence(content);
  if (!fence) {
    return content;
  }

  const newline = content.endsWith('\n') ? '' : '\n';
  return `${content}${newline}${fence.marker}`;
}

function findUnclosedFence(content: string): { marker: string } | null {
  let openFence: { marker: string } | null = null;
  const lines = content.split('\n');

  for (const line of lines) {
    const match = /^(\s{0,3})(`{3,}|~{3,})/.exec(line);
    if (!match) {
      continue;
    }

    const marker = match[2][0].repeat(match[2].length);
    if (!openFence) {
      openFence = { marker };
      continue;
    }

    if (marker[0] === openFence.marker[0] && marker.length >= openFence.marker.length) {
      openFence = null;
    }
  }

  return openFence;
}
