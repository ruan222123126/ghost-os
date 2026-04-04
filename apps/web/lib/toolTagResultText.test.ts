import { filterToolTagResultToLoadedTools } from './toolTagResultText';

const TOOL_TAG_RESULT_PREFIX = '[TOOL_TAG_RESULT]\n';

describe('toolTagResultText', () => {
  it('keeps only loaded tfind items in tool-tag result payload', () => {
    const raw = `${TOOL_TAG_RESULT_PREFIX}${JSON.stringify({
      tool: 'tfind',
      output: {
        action: 'list',
        items: [
          { name: 'browser_control', status: 'active', available_now: true },
          { name: 'computer_use', status: 'pending', available_next_turn: true },
          { name: 'web_search', status: 'expired', available_now: false },
          { name: 'rss_fetch', status: 'unloaded' },
        ],
      },
    })}`;

    const filtered = filterToolTagResultToLoadedTools(raw);
    const payload = parseToolTagResult(filtered);

    expect(payload?.output?.items).toEqual([
      { name: 'browser_control', status: 'active', available_now: true },
      { name: 'computer_use', status: 'pending', available_next_turn: true },
    ]);
  });

  it('keeps non-tfind tool-tag results untouched', () => {
    const raw = `${TOOL_TAG_RESULT_PREFIX}${JSON.stringify({
      tool: 'web_search',
      output: {
        items: [{ title: 'OpenAI' }],
      },
    })}`;

    expect(filterToolTagResultToLoadedTools(raw)).toBe(raw);
  });
});

function parseToolTagResult(text: string): {
  output?: {
    items?: unknown[];
  };
} | null {
  const payload = text.replace(TOOL_TAG_RESULT_PREFIX, '').trim();
  return JSON.parse(payload) as { output?: { items?: unknown[] } };
}
