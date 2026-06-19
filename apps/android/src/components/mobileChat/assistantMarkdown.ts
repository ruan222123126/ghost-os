const DEFAULT_CODE_LANGUAGE = "text";
const LANGUAGE_NAME_PATTERN = /language-([^\s]+)/i;
const LANGUAGE_SANITIZE_PATTERN = /[^a-z0-9#+.-]/g;

const LANGUAGE_ALIASES: Record<string, string> = {
  cjs: "javascript",
  js: "javascript",
  jsx: "jsx",
  mjs: "javascript",
  py: "python",
  rb: "ruby",
  rs: "rust",
  sh: "bash",
  shell: "bash",
  ts: "typescript",
  tsx: "tsx",
  yml: "yaml",
  zsh: "bash",
};

interface HighlightToken {
  html: string;
  token: string;
}

export function extractCodeLanguage(className: string | undefined): string {
  const match = className?.match(LANGUAGE_NAME_PATTERN);
  return normalizeCodeLanguage(match?.[1]);
}

export function formatCodeLanguageLabel(language: string): string {
  const trimmed = language.trim();
  return trimmed ? trimmed.toUpperCase() : DEFAULT_CODE_LANGUAGE.toUpperCase();
}

export function highlightCodeBlockHTML(code: string): string {
  const tokens: HighlightToken[] = [];
  let html = protectCodeLiterals(escapeCodeHTML(code), (tokenHTML) => createHighlightToken(tokens, tokenHTML));

  html = applyHighlight(html, /\b(import|from|export|default|return|if|else|try|catch)\b/g, tokens, "control");
  html = applyHighlight(html, /\b(async|await|function|const|let|var)\b/g, tokens, "declaration");
  html = html.replace(/(&lt;\/?)([A-Za-z0-9]+)/g, (_match, prefix: string, tag: string) => (
    `${prefix}${createHighlightToken(tokens, wrapSyntax("tag", tag))}`
  ));
  html = html.replace(/(\s)([a-zA-Z-]+)(=)/g, (_match, prefix: string, name: string, suffix: string) => (
    `${prefix}${createHighlightToken(tokens, wrapSyntax("attribute", name))}${suffix}`
  ));
  html = applyHighlight(html, /\b([A-Z][a-zA-Z0-9_]*)\b/g, tokens, "type");
  html = applyHighlight(html, /\b([a-z_]\w*)(?=\s*\()/g, tokens, "function");
  html = applyHighlight(html, /\b(\d+)\b/g, tokens, "number");

  return restoreHighlightTokens(html, tokens);
}

function normalizeCodeLanguage(language?: string): string {
  const sanitized = (language || "")
    .trim()
    .toLowerCase()
    .replace(LANGUAGE_SANITIZE_PATTERN, "");
  return sanitized ? LANGUAGE_ALIASES[sanitized] || sanitized : DEFAULT_CODE_LANGUAGE;
}

function escapeCodeHTML(code: string): string {
  return code
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

function protectCodeLiterals(html: string, createToken: (html: string) => string): string {
  let protectedHTML = "";
  let index = 0;

  while (index < html.length) {
    const char = html[index];
    const next = html[index + 1];

    if (char === "/" && next === "/") {
      const end = html.indexOf("\n", index);
      const boundary = end === -1 ? html.length : end;
      protectedHTML += createToken(wrapSyntax("comment", html.slice(index, boundary)));
      index = boundary;
      continue;
    }

    if (char === "'" || char === "\"" || char === "`") {
      const end = findStringEnd(html, index, char);
      protectedHTML += createToken(wrapSyntax("string", html.slice(index, end)));
      index = end;
      continue;
    }

    protectedHTML += char;
    index += 1;
  }

  return protectedHTML;
}

function findStringEnd(html: string, start: number, quote: string): number {
  let index = start + 1;
  while (index < html.length) {
    if (html[index] === "\\") {
      index += 2;
      continue;
    }
    if (html[index] === quote) {
      return index + 1;
    }
    index += 1;
  }
  return html.length;
}

function applyHighlight(html: string, pattern: RegExp, tokens: HighlightToken[], className: string): string {
  return html.replace(pattern, (match: string) => createHighlightToken(tokens, wrapSyntax(className, match)));
}

function wrapSyntax(className: string, value: string): string {
  return `<span class="assistant-syntax assistant-syntax-${className}">${value}</span>`;
}

function createHighlightToken(tokens: HighlightToken[], html: string): string {
  const token = String.fromCharCode(0xe000 + tokens.length);
  tokens.push({ html, token });
  return token;
}

function restoreHighlightTokens(html: string, tokens: readonly HighlightToken[]): string {
  let restored = html;
  for (let index = tokens.length - 1; index >= 0; index -= 1) {
    const { token, html: tokenHTML } = tokens[index];
    restored = restored.split(token).join(tokenHTML);
  }
  return restored;
}
