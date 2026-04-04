import {
  consumeToolTagStreamChunk,
  createToolTagStreamState,
  isPureToolTagDocument,
  parseToolTagText,
  stripToolTagCalls,
} from './toolTagText';

describe('lib/toolTagText', () => {
  it('parses consecutive tool tags and strips visible text', () => {
    const text = 'prefix<t:1>{"query":"OpenAI"}</t><t:2>{"script":"echo ok"}</t>suffix';
    const parsed = parseToolTagText(text);

    expect(parsed.hasToolTags).toBe(true);
    expect(parsed.visibleText).toBe('prefixsuffix');
    expect(parsed.calls).toEqual([
      { toolId: '1', argsText: '{"query":"OpenAI"}' },
      { toolId: '2', argsText: '{"script":"echo ok"}' },
    ]);
    expect(stripToolTagCalls(text)).toBe('prefixsuffix');
  });

  it('emits tool_open immediately after entering CAPTURE_ARGS', () => {
    const state = createToolTagStreamState();

    const first = consumeToolTagStreamChunk(state, '<t:2>');
    expect(first.visibleText).toBe('');
    expect(first.events).toEqual([
      { type: 'tool_open', callSeq: 1, toolId: '2' },
    ]);

    const second = consumeToolTagStreamChunk(state, '{"script":"echo ok"}');
    expect(second.events).toEqual([
      { type: 'tool_args', callSeq: 1, argsDelta: '{"script":"echo ok"}' },
    ]);

    const third = consumeToolTagStreamChunk(state, '</t>');
    expect(third.events).toEqual([
      { type: 'tool_close', callSeq: 1, toolId: '2', argsText: '{"script":"echo ok"}' },
    ]);
  });

  it('keeps text and tool units in original stream order', () => {
    const state = createToolTagStreamState();
    const consumed = consumeToolTagStreamChunk(
      state,
      'alpha<t:1>{"q":"x"}</t>omega',
    );

    expect(consumed.units).toEqual([
      { type: 'text', text: 'alpha' },
      { type: 'tool_open', callSeq: 1, toolId: '1' },
      { type: 'tool_args', callSeq: 1, argsDelta: '{"q":"x"}' },
      { type: 'tool_close', callSeq: 1, toolId: '1', argsText: '{"q":"x"}' },
      { type: 'text', text: 'omega' },
    ]);
  });

  it('treats </t> inside JSON string as argument text', () => {
    const state = createToolTagStreamState();
    const consumed = consumeToolTagStreamChunk(
      state,
      '<t:1>{"text":"literal </t> marker"}</t>',
    );

    expect(consumed.events).toEqual([
      { type: 'tool_open', callSeq: 1, toolId: '1' },
      { type: 'tool_args', callSeq: 1, argsDelta: '{"text":"literal </t> marker"}' },
      { type: 'tool_close', callSeq: 1, toolId: '1', argsText: '{"text":"literal </t> marker"}' },
    ]);
  });

  it('finalize closes unterminated tool tags at stream end', () => {
    const state = createToolTagStreamState();
    consumeToolTagStreamChunk(state, '<t:1>{"query":"OpenAI"');

    const finalized = consumeToolTagStreamChunk(state, '', true);
    expect(finalized.events).toEqual([
      { type: 'tool_close', callSeq: 1, toolId: '1', argsText: '{"query":"OpenAI"' },
    ]);
  });

  it('does not treat non-numeric ids as tool tags', () => {
    const text = '<t:x>{"query":"OpenAI"}</t>';
    const parsed = parseToolTagText(text);

    expect(parsed.hasToolTags).toBe(false);
    expect(parsed.calls).toEqual([]);
    expect(parsed.visibleText).toBe(text);
    expect(isPureToolTagDocument(text)).toBe(false);
  });
});
