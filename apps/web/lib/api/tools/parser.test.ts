import { parseToolPayload, parseToolPayloadList } from './parser';

describe('lib/api/tools/parser', () => {
  it('parses tool list payload', () => {
    const payload = [
      {
        name: 'script_exec',
        enabled: true,
        prompt_override: 'use with named parameters',
        sandbox_memory_mb: 384,
        input_schema: {
          type: 'object',
          properties: {
            command: { type: 'string' },
          },
        },
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
        sandbox_memory_mb: 384,
        input_schema: {
          type: 'object',
          properties: {
            command: { type: 'string' },
          },
        },
      },
      {
        name: 'web_search',
        enabled: false,
        prompt_override: undefined,
        sandbox_memory_mb: undefined,
        input_schema: undefined,
      },
    ]);
  });

  it('parses single tool payload', () => {
    const payload = {
      name: 'script_exec',
      enabled: true,
      prompt_override: 'custom prompt',
      sandbox_memory_mb: 320,
      input_schema: { type: 'object' },
    };

    expect(parseToolPayload(payload)).toEqual({
      name: 'script_exec',
      enabled: true,
      prompt_override: 'custom prompt',
      sandbox_memory_mb: 320,
      input_schema: { type: 'object' },
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

  it('treats null input_schema as undefined', () => {
    expect(parseToolPayload({
      name: 'web_search',
      enabled: true,
      input_schema: null,
    })).toEqual({
      name: 'web_search',
      enabled: true,
      prompt_override: undefined,
      sandbox_memory_mb: undefined,
      input_schema: undefined,
    });
  });

  it('treats boolean input_schema as undefined', () => {
    expect(parseToolPayload({
      name: 'web_search',
      enabled: true,
      input_schema: true,
    })).toEqual({
      name: 'web_search',
      enabled: true,
      prompt_override: undefined,
      sandbox_memory_mb: undefined,
      input_schema: undefined,
    });
  });
});
