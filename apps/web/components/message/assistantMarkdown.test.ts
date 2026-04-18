import {
  extractCodeLanguage,
  formatCodeLanguageLabel,
  shouldRenderAssistantMarkdown,
} from './assistantMarkdown';

describe('shouldRenderAssistantMarkdown', () => {
  it('returns false for plain text', () => {
    expect(shouldRenderAssistantMarkdown('just a normal reply')).toBe(false);
    expect(shouldRenderAssistantMarkdown('Version 1.0.0 released today.')).toBe(false);
  });

  it('returns true for heading and list syntax', () => {
    expect(shouldRenderAssistantMarkdown('# Title\n\ncontent')).toBe(true);
    expect(shouldRenderAssistantMarkdown('- item one\n- item two')).toBe(true);
  });

  it('returns true for fenced and inline code', () => {
    expect(shouldRenderAssistantMarkdown('```go\nfmt.Println("ok")\n```')).toBe(true);
    expect(shouldRenderAssistantMarkdown('Use `pnpm test` to run tests.')).toBe(true);
  });

  it('returns true for links and emphasis', () => {
    expect(shouldRenderAssistantMarkdown('[Ghost-OS](https://example.com)')).toBe(true);
    expect(shouldRenderAssistantMarkdown('**important** result')).toBe(true);
  });
});

describe('extractCodeLanguage', () => {
  it('normalizes aliases and falls back to text', () => {
    expect(extractCodeLanguage('language-ts')).toBe('typescript');
    expect(extractCodeLanguage('foo language-PY bar')).toBe('python');
    expect(extractCodeLanguage('language-bash')).toBe('bash');
    expect(extractCodeLanguage('')).toBe('text');
    expect(extractCodeLanguage(undefined)).toBe('text');
  });

  it('sanitizes unusual language values', () => {
    expect(extractCodeLanguage('language-C++')).toBe('c++');
    expect(extractCodeLanguage('language-unknown$lang')).toBe('unknownlang');
  });
});

describe('formatCodeLanguageLabel', () => {
  it('renders uppercase labels', () => {
    expect(formatCodeLanguageLabel('typescript')).toBe('TYPESCRIPT');
    expect(formatCodeLanguageLabel('')).toBe('TEXT');
  });
});
