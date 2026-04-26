import type { StreamingAssistantSegment, StreamingToolState } from '@/lib/types';
import {
  hasAssistantStreamedVisibleText,
  shouldAutoCollapseThinkingPanel,
  shouldExpandThinkingPanelByDefault,
  shouldShowThinkingIndicator,
} from './thinkingState';

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
      streamingThinkingText: '',
      streamingAssistantSegments: [],
      streamingTools: [],
    })).toBe(false);
  });

  it('returns true when loading without text output and tool activity', () => {
    expect(shouldShowThinkingIndicator({
      loading: true,
      streamingThinkingText: '',
      streamingAssistantSegments: [],
      streamingTools: [],
    })).toBe(true);
  });

  it('returns false when assistant has streamed text', () => {
    expect(shouldShowThinkingIndicator({
      loading: true,
      streamingThinkingText: '',
      streamingAssistantSegments: [buildAssistantSegment('hello')],
      streamingTools: [],
    })).toBe(false);
  });

  it('returns false when streaming thinking text exists', () => {
    expect(shouldShowThinkingIndicator({
      loading: true,
      streamingThinkingText: 'analyzing',
      streamingAssistantSegments: [],
      streamingTools: [],
    })).toBe(false);
  });

  it('returns false when any tool is running', () => {
    for (const status of ['pending', 'running', 'in_progress']) {
      expect(shouldShowThinkingIndicator({
        loading: true,
        streamingThinkingText: '',
        streamingAssistantSegments: [],
        streamingTools: [buildStreamingTool(status)],
      })).toBe(false);
    }
  });

  it('returns true when tools are finished', () => {
    expect(shouldShowThinkingIndicator({
      loading: true,
      streamingThinkingText: '',
      streamingAssistantSegments: [],
      streamingTools: [buildStreamingTool('success'), buildStreamingTool('error')],
    })).toBe(true);
  });
});

describe('hasAssistantStreamedVisibleText', () => {
  it('returns true when any segment has non-empty text', () => {
    expect(hasAssistantStreamedVisibleText([
      buildAssistantSegment(''),
      buildAssistantSegment('hello'),
    ])).toBe(true);
  });

  it('returns false when all segments are blank', () => {
    expect(hasAssistantStreamedVisibleText([
      buildAssistantSegment(''),
      buildAssistantSegment('   '),
    ])).toBe(false);
  });
});

describe('thinking panel transition', () => {
  it('expands by default when thinking appears for the first time', () => {
    expect(shouldExpandThinkingPanelByDefault(true, false)).toBe(true);
    expect(shouldExpandThinkingPanelByDefault(true, true)).toBe(false);
    expect(shouldExpandThinkingPanelByDefault(false, false)).toBe(false);
  });

  it('auto-collapses on first assistant text token after thinking starts', () => {
    expect(shouldAutoCollapseThinkingPanel({
      hasThinkingText: true,
      hasAssistantText: true,
      hadAssistantText: false,
      alreadyAutoCollapsed: false,
    })).toBe(true);
  });

  it('does not auto-collapse without thinking text or after first collapse', () => {
    expect(shouldAutoCollapseThinkingPanel({
      hasThinkingText: false,
      hasAssistantText: true,
      hadAssistantText: false,
      alreadyAutoCollapsed: false,
    })).toBe(false);
    expect(shouldAutoCollapseThinkingPanel({
      hasThinkingText: true,
      hasAssistantText: true,
      hadAssistantText: true,
      alreadyAutoCollapsed: false,
    })).toBe(false);
    expect(shouldAutoCollapseThinkingPanel({
      hasThinkingText: true,
      hasAssistantText: true,
      hadAssistantText: false,
      alreadyAutoCollapsed: true,
    })).toBe(false);
  });
});
