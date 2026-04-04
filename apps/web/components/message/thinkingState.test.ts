import type { StreamingAssistantSegment, StreamingToolState } from '@/lib/types';
import { shouldShowThinkingIndicator } from './thinkingState';

function buildStreamingTool(toolStatus?: string): StreamingToolState {
  return {
    id: 'tool-1',
    content: 'tool output',
    toolStatus,
  };
}

function buildAssistantSegment(content: string): StreamingAssistantSegment {
  return {
    id: 'assistant-segment-1',
    content,
  };
}

describe('shouldShowThinkingIndicator', () => {
  it('returns false when loading is false', () => {
    expect(shouldShowThinkingIndicator({
      loading: false,
      streamingAssistantSegments: [],
      streamingTools: [],
    })).toBe(false);
  });

  it('returns true when loading without text output and tool activity', () => {
    expect(shouldShowThinkingIndicator({
      loading: true,
      streamingAssistantSegments: [],
      streamingTools: [],
    })).toBe(true);
  });

  it('returns false when assistant has streamed text', () => {
    expect(shouldShowThinkingIndicator({
      loading: true,
      streamingAssistantSegments: [buildAssistantSegment('hello')],
      streamingTools: [],
    })).toBe(false);
  });

  it('returns false when any tool is running', () => {
    for (const status of ['pending', 'running', 'in_progress']) {
      expect(shouldShowThinkingIndicator({
        loading: true,
        streamingAssistantSegments: [],
        streamingTools: [buildStreamingTool(status)],
      })).toBe(false);
    }
  });

  it('returns true when tools are finished', () => {
    expect(shouldShowThinkingIndicator({
      loading: true,
      streamingAssistantSegments: [],
      streamingTools: [buildStreamingTool('success'), buildStreamingTool('error')],
    })).toBe(true);
  });
});
