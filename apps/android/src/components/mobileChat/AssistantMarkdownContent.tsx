import { memo, useDeferredValue, useMemo } from "react";
import MarkdownRender, { setCustomComponents } from "markstream-react";
import type { CustomComponentMap, NodeComponentProps } from "markstream-react";
import { formatAssistantTextForDisplay } from "../../../../shared/assistantTextSpacing";
import { MessageCopyButton } from "./MessageCopyButton";

const ASSISTANT_MARKDOWN_CUSTOM_ID = "ghost-os-mobile-assistant-markdown";

interface AssistantMarkdownContentProps {
  content: string;
  final?: boolean;
  showCopyButton?: boolean;
}

interface AssistantCodeBlockNode {
  code?: string;
  content?: string;
  language?: string;
  lang?: string;
}

interface AssistantCodeBlockProps extends NodeComponentProps<AssistantCodeBlockNode> {
  showCopyButton?: boolean;
}

const CODE_BLOCK_DEFAULT_LANGUAGE = "text";
const FENCED_CODE_BLOCK_PATTERN = /(^|\n)(`{3,}|~{3,})[^\n]*(?=\n|$)/g;

function AssistantCodeBlockCard({
  node,
  showCopyButton = true,
}: AssistantCodeBlockProps) {
  const code = String(node.code ?? node.content ?? "");
  const language = formatCodeLanguage(node.language ?? node.lang);

  return (
    <div className="assistant-code-block-card">
      <div className="assistant-code-block-header">
        <span className="assistant-code-block-language">{language}</span>
        {showCopyButton ? <MessageCopyButton text={code} variant="code" /> : null}
      </div>
      <pre className="assistant-code-block-pre" data-language={language}>
        <code>{code}</code>
      </pre>
    </div>
  );
}

const assistantMarkdownComponents = {
  code_block: AssistantCodeBlockCard,
} as unknown as CustomComponentMap;

setCustomComponents(ASSISTANT_MARKDOWN_CUSTOM_ID, assistantMarkdownComponents);

function formatCodeLanguage(language: string | undefined): string {
  const normalized = language?.trim().split(/\s+/)[0]?.toLowerCase();
  return normalized || CODE_BLOCK_DEFAULT_LANGUAGE;
}

function AssistantMarkdownContentBase({
  content,
  final = true,
  showCopyButton = true,
}: AssistantMarkdownContentProps) {
  const deferredContent = useDeferredValue(content);
  const displayContent = useMemo(
    () => prepareAssistantMarkdownForRender(deferredContent, final),
    [deferredContent, final],
  );

  return (
    <div className={final ? "assistant-markdown" : "assistant-markdown assistant-markdown-streaming"}>
      <MarkdownRender
        codeBlockProps={{
          enableFontSizeControl: false,
          showCollapseButton: false,
          showCopyButton,
          showExpandButton: true,
          showFontSizeButtons: false,
          showPreviewButton: false,
        }}
        content={displayContent}
        customId={ASSISTANT_MARKDOWN_CUSTOM_ID}
        d2Props={{ showCopyButton }}
        fade={false}
        final={final}
        htmlPolicy="safe"
        infographicProps={{ showCopyButton }}
        mermaidProps={{ showCopyButton }}
        showTooltips={false}
        typewriter={false}
      />
    </div>
  );
}

export const AssistantMarkdownContent = memo(AssistantMarkdownContentBase);

function prepareAssistantMarkdownForRender(content: string, final: boolean): string {
  const displayContent = formatAssistantTextForDisplay(content);
  if (final) {
    return displayContent;
  }
  return closeDanglingMarkdownBlocks(displayContent);
}

function closeDanglingMarkdownBlocks(content: string): string {
  const openFence = findDanglingFence(content);
  if (!openFence) {
    return content;
  }
  return `${content}\n${openFence}`;
}

function findDanglingFence(content: string): string | null {
  const stack: string[] = [];
  for (const match of content.matchAll(FENCED_CODE_BLOCK_PATTERN)) {
    const fence = match[2];
    const marker = fence[0];
    if (stack.length > 0 && stack[stack.length - 1]?.startsWith(marker)) {
      stack.pop();
      continue;
    }
    stack.push(fence);
  }
  return stack.length > 0 ? stack[stack.length - 1] : null;
}
