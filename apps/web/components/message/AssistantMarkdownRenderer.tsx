import type { FC } from 'react';
import MarkdownRender from 'markstream-react';

interface AssistantMarkdownRendererProps {
  content: string;
  final?: boolean;
  showCopyButton?: boolean;
}

export const AssistantMarkdownRenderer: FC<AssistantMarkdownRendererProps> = ({
  content,
  final = true,
  showCopyButton = true,
}) => (
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
);
