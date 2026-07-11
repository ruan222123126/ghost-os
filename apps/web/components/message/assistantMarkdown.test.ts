import { shouldRenderAssistantMarkdown } from './assistantMarkdown';

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

  it('returns true for double-dollar math expressions', () => {
    expect(shouldRenderAssistantMarkdown('如果意思是 9：$$\\frac{6}{2}(1+2)$$')).toBe(true);
    expect(shouldRenderAssistantMarkdown('如果意思是 1：$$\\frac{6}{2(1+2)}$$')).toBe(true);
    expect(shouldRenderAssistantMarkdown('$$\\frac{6}{2}(1+2)$$')).toBe(true);
  });

  it('does not treat ordinary dollar amounts as markdown', () => {
    expect(shouldRenderAssistantMarkdown('Price moved from $6 to $7 today.')).toBe(false);
  });
});
