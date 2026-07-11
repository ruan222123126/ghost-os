import { memo } from "react";
import MarkdownRender from "markstream-react";

interface AssistantMarkdownContentProps {
  content: string;
  final?: boolean;
  showCopyButton?: boolean;
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
