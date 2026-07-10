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
  return stabilizeStreamingMarkdown(displayContent);
}

function stabilizeStreamingMarkdown(content: string): string {
  const openFence = findDanglingFence(content);
  if (openFence) {
    return `${content}\n${openFence}`;
  }
  return completeTrailingMarkdownTable(content);
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

function completeTrailingMarkdownTable(content: string): string {
  const lines = content.split("\n");
  const lastLineIndex = lines.length - 1;
  const lastCells = parsePipeCells(lines[lastLineIndex] ?? "", 1);
  if (!lastCells) {
    return content;
  }

  const previousLineIndex = lastLineIndex - 1;
  const previousCells = previousLineIndex >= 0 ? parseTableCells(lines[previousLineIndex] ?? "") : null;
  if (!previousCells) {
    if (lastCells.length < 2 || !hasClosedTableRow(lines[lastLineIndex] ?? "")) {
      return content;
    }
    lines.push(buildTableDelimiter(lastCells.length));
    return lines.join("\n");
  }

  if (isPossibleTableDelimiter(lastCells)) {
    lines[lastLineIndex] = buildTableDelimiter(previousCells.length);
    return lines.join("\n");
  }
  if (isPossibleTableDelimiter(previousCells)) {
    lines[lastLineIndex] = buildNormalizedTableRow(lastCells, previousCells.length);
  }
  return lines.join("\n");
}

function parseTableCells(line: string): string[] | null {
  return parsePipeCells(line, 2);
}

function parsePipeCells(line: string, minimumCellCount: number): string[] | null {
  const trimmed = line.trim();
  if (!trimmed.startsWith("|")) {
    return null;
  }
  const body = trimmed.slice(1, trimmed.endsWith("|") ? -1 : undefined);
  const cells = splitUnescapedPipes(body).map((cell) => cell.trim());
  return cells.length >= minimumCellCount ? cells : null;
}

function splitUnescapedPipes(value: string): string[] {
  const cells: string[] = [];
  let current = "";
  let escaped = false;
  for (const character of value) {
    if (character === "|" && !escaped) {
      cells.push(current);
      current = "";
    } else {
      current += character;
    }
    escaped = character === "\\" && !escaped;
    if (character !== "\\") {
      escaped = false;
    }
  }
  cells.push(current);
  return cells;
}

function hasClosedTableRow(line: string): boolean {
  return line.trim().endsWith("|");
}

function isPossibleTableDelimiter(cells: string[]): boolean {
  return cells.some((cell) => cell.includes("-"))
    && cells.every((cell) => cell === "" || /^:?-+:?$/.test(cell));
}

function buildTableDelimiter(columnCount: number): string {
  return `| ${Array.from({ length: columnCount }, () => "---").join(" | ")} |`;
}

function buildNormalizedTableRow(cells: string[], columnCount: number): string {
  const normalizedCells = Array.from({ length: columnCount }, (_, index) => cells[index] ?? "");
  return `| ${normalizedCells.join(" | ")} |`;
}
