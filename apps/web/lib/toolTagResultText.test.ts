import { filterToolTagResultToLoadedTools } from './toolTagResultText';

const TOOL_TAG_RESULT_PREFIX = '[TOOL_TAG_RESULT]\n';

describe('toolTagResultText', () => {
  it('keeps only loaded sfind items in tool-tag result payload', () => {
    const raw = `${TOOL_TAG_RESULT_PREFIX}${JSON.stringify({
      tool: 'sfind',
      output: {
        action: 'list',
        items: [
          { name: 'release_flow', status: 'active', available_now: true },
          { name: 'ship_checklist', status: 'pending', available_next_turn: true },
          { name: 'incident_triage', status: 'expired', available_now: false },
          { name: 'deploy_runbook', status: 'unloaded' },
        ],
      },
    })}`;

    const filtered = filterToolTagResultToLoadedTools(raw);
    const payload = parseToolTagResult(filtered);

    expect(payload?.output?.items).toEqual([
      { name: 'release_flow', status: 'active', available_now: true },
      { name: 'ship_checklist', status: 'pending', available_next_turn: true },
    ]);
  });

  it('keeps non-sfind tool-tag results untouched', () => {
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
