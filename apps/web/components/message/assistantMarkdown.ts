const DEFAULT_CODE_LANGUAGE = 'text';

const BLOCK_MARKDOWN_PATTERNS = [
  /(^|\n)\s{0,3}#{1,6}[ \t]+\S/,
  /(^|\n)\s{0,3}>[ \t]?\S/,
  /(^|\n)\s{0,3}(?:[-+*][ \t]+\S|\d+\.[ \t]+\S)/,
  /(^|\n)\s*```/,
  /(^|\n)\s*\|.+\|/,
  /(^|\n)\s{0,3}(?:---|\*\*\*|___)\s*($|\n)/,
] as const;

const INLINE_MARKDOWN_PATTERNS = [
  /`[^`\n]+`/,
  /\[[^\]]+\]\([^)]+\)/,
  /\*\*[^*\n]+\*\*/,
  /~~[^~\n]+~~/,
] as const;

const MARKDOWN_TOKENS = ['`', '[', '#', '*', '-', '+', '>', '|', '~', '\n1.'] as const;
const LANGUAGE_NAME_PATTERN = /language-([^\s]+)/i;
const LANGUAGE_SANITIZE_PATTERN = /[^a-z0-9#+.-]/g;

const LANGUAGE_ALIASES: Record<string, string> = {
  cjs: 'javascript',
  js: 'javascript',
  jsx: 'jsx',
  mjs: 'javascript',
  ts: 'typescript',
  tsx: 'tsx',
  py: 'python',
  rb: 'ruby',
  rs: 'rust',
  sh: 'bash',
  shell: 'bash',
  zsh: 'bash',
  yml: 'yaml',
};

export function shouldRenderAssistantMarkdown(content: string): boolean {
  const trimmed = content.trim();
  if (!trimmed || !containsMarkdownTokens(trimmed)) {
    return false;
  }
  return matchesMarkdownPattern(trimmed, BLOCK_MARKDOWN_PATTERNS)
    || matchesMarkdownPattern(trimmed, INLINE_MARKDOWN_PATTERNS);
}

export function extractCodeLanguage(className?: string): string {
  if (!className) {
    return DEFAULT_CODE_LANGUAGE;
  }
  const matched = LANGUAGE_NAME_PATTERN.exec(className);
  return normalizeCodeLanguage(matched?.[1]);
}

export function formatCodeLanguageLabel(language: string): string {
  const trimmed = language.trim();
  if (!trimmed) {
    return DEFAULT_CODE_LANGUAGE.toUpperCase();
  }
  return trimmed.toUpperCase();
}

function normalizeCodeLanguage(language?: string): string {
  const sanitized = (language || '')
    .trim()
    .toLowerCase()
    .replace(LANGUAGE_SANITIZE_PATTERN, '');
  if (!sanitized) {
    return DEFAULT_CODE_LANGUAGE;
  }
  return LANGUAGE_ALIASES[sanitized] || sanitized;
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
