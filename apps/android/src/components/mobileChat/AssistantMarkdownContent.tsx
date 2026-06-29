import { memo } from "react";
import MarkdownRender, { setCustomComponents } from "markstream-react";
import type { CustomComponentMap, NodeComponentProps } from "markstream-react";
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
  return (
    <div className="assistant-markdown">
      <MarkdownRender
        codeBlockProps={{
          enableFontSizeControl: false,
          showCollapseButton: false,
          showCopyButton,
          showExpandButton: true,
          showFontSizeButtons: false,
          showPreviewButton: false,
        }}
        content={content}
        customId={ASSISTANT_MARKDOWN_CUSTOM_ID}
        d2Props={{ showCopyButton }}
        fade={false}
        final={final}
        htmlPolicy="safe"
        infographicProps={{ showCopyButton }}
        mermaidProps={{ showCopyButton }}
        showTooltips={false}
        typewriter={!final}
      />
    </div>
  );
}

export const AssistantMarkdownContent = memo(AssistantMarkdownContentBase);
