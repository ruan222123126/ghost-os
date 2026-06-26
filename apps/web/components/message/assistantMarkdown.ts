const BLOCK_MARKDOWN_PATTERNS = [
  /(^|\n)\s{0,3}#{1,6}[ \t]+\S/,
  /(^|\n)\s{0,3}>[ \t]?\S/,
  /(^|\n)\s{0,3}(?:[-+*][ \t]+\S|\d+\.[ \t]+\S)/,
  /(^|\n)\s*```/,
  /(^|\n)\s*\|.+\|/,
  /(^|\n)\s{0,3}(?:---|\*\*\*|___)\s*($|\n)/,
  /(^|\n)\s{0,3}\$\$[\s\S]+?\$\$/,
] as const;

const INLINE_MARKDOWN_PATTERNS = [
  /`[^`\n]+`/,
  /\[[^\]]+\]\([^)]+\)/,
  /\*\*[^*\n]+\*\*/,
  /~~[^~\n]+~~/,
  /\$\$[^$\n]+\$\$/,
] as const;

const MARKDOWN_TOKENS = ['`', '[', '#', '*', '-', '+', '>', '|', '~', '$', '\n1.'] as const;

export function shouldRenderAssistantMarkdown(content: string): boolean {
  const trimmed = content.trim();
  if (!trimmed || !containsMarkdownTokens(trimmed)) {
    return false;
  }
  return matchesMarkdownPattern(trimmed, BLOCK_MARKDOWN_PATTERNS)
    || matchesMarkdownPattern(trimmed, INLINE_MARKDOWN_PATTERNS);
}

function containsMarkdownTokens(content: string): boolean {
  for (const token of MARKDOWN_TOKENS) {
    if (content.includes(token)) {
      return true;
    }
  }
  return false;
}

function matchesMarkdownPattern(content: string, patterns: readonly RegExp[]): boolean {
  for (const pattern of patterns) {
    if (pattern.test(content)) {
      return true;
    }
  }
  return false;
}
