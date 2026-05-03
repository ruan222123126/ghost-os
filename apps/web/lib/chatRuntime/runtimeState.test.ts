import { createChatRuntimeStateFromDraft } from './runtimeState';

describe('chatRuntime/runtimeState', () => {
  it('rehydrates runtime buffers and tool lookups from turn_draft', () => {
    const runtime = createChatRuntimeStateFromDraft({
      trace_id: 'trace-1',
      turn: 2,
      assistant_segments: [
        { id: 'stream-segment:assistant:1', content: 'part 1' },
        { id: 'stream-segment:assistant:2', content: 'part 2' },
      ],
      thinking_segments: [{ id: 'stream-segment:thinking:1', content: 'analysis' }],
      tools: [{
        id: 'stream-tool:trace-1:preview:1:index:0',
        content: '{"command":"pwd"}',
        tool_name: 'bash_exec',
        tool_status: 'pending',
      }],
      item_order: [
        'thinking:stream-segment:thinking:1',
        'tool:stream-tool:trace-1:preview:1:index:0',
      ],
    }, 'session-1');

    expect(runtime.assistantBuffer).toBe('part 1part 2');
    expect(runtime.thinkingBuffers).toEqual(['analysis']);
    expect(runtime.thinkingActiveSegmentIndex).toBe(-1);
    expect(runtime.previewStructuredToolMessageIdsByIndex.get(0)).toBe('stream-tool:trace-1:preview:1:index:0');
    expect(runtime.pendingPreviewQueue).toEqual(['stream-tool:trace-1:preview:1:index:0']);
  });
});
