import { parsePresetPayload, parsePresetPayloadList } from './parser';

describe('lib/api/presets/parser', () => {
  it('parses preset list payload', () => {
    const payload = [
      {
        id: 'preset-a',
        name: 'Default',
        tool_allowlist: ['script_exec', 'web_search'],
        prompt_refs: {
          rule: 'rule-card',
          core_job: 'core-card',
          context: ['context-a', 'context-b'],
        },
      },
    ];

    expect(parsePresetPayloadList(payload)).toEqual([
      {
        id: 'preset-a',
        name: 'Default',
        tool_allowlist: ['script_exec', 'web_search'],
        prompt_refs: {
          rule: 'rule-card',
          core_job: 'core-card',
          memory: undefined,
          context: ['context-a', 'context-b'],
        },
      },
    ]);
  });

  it('parses single preset payload', () => {
    expect(parsePresetPayload({
      id: 'preset-a',
      name: 'Default',
      tool_allowlist: [],
      prompt_refs: {},
    })).toEqual({
      id: 'preset-a',
      name: 'Default',
      tool_allowlist: [],
      prompt_refs: {
        rule: undefined,
        core_job: undefined,
        memory: undefined,
        context: undefined,
      },
    });
  });

  it('rejects invalid prompt_refs shape', () => {
    expect(() => {
      parsePresetPayload({
        id: 'preset-a',
        name: 'Default',
        tool_allowlist: [],
        prompt_refs: 'rule-card',
      });
    }).toThrow('preset payload.prompt_refs');

    expect(() => {
      parsePresetPayload({
        id: 'preset-a',
        name: 'Default',
        tool_allowlist: [],
        prompt_refs: {
          context: 'context-a',
        },
      });
    }).toThrow('preset payload.prompt_refs.context');
  });
});
