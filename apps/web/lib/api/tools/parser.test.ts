import { parseToolPayload, parseToolPayloadList } from './parser';

describe('lib/api/tools/parser', () => {
  it('parses tool list payload', () => {
    const payload = [
      {
        name: 'script_exec',
        enabled: true,
        prompt_override: 'use with named parameters',
      },
      {
        name: 'web_search',
        enabled: false,
      },
    ];

    expect(parseToolPayloadList(payload)).toEqual([
      {
        name: 'script_exec',
        enabled: true,
        prompt_override: 'use with named parameters',
      },
      {
        name: 'web_search',
        enabled: false,
        prompt_override: undefined,
      },
    ]);
  });

  it('parses single tool payload', () => {
    const payload = {
      name: 'script_exec',
      enabled: true,
      prompt_override: 'custom prompt',
    };

    expect(parseToolPayload(payload)).toEqual({
      name: 'script_exec',
      enabled: true,
      prompt_override: 'custom prompt',
    });
  });

  it('throws on invalid enabled type', () => {
    expect(() => {
      parseToolPayload({
        name: 'script_exec',
        enabled: 'yes',
      });
    }).toThrow('tool payload.enabled');
  });
});
