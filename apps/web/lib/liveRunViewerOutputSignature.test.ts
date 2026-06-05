import { buildAssistantMessage } from '@/lib/chatMessages';
import { buildLiveRunViewerOutputSignature } from './liveRunViewerOutputSignature';
import type { TaskRunCardOutput } from '@/lib/taskRunViewerOutput';

describe('liveRunViewerOutputSignature', () => {
  it('changes when streaming assistant text grows', () => {
    const initial = buildLiveRunViewerOutputSignature('card-1', outputWithStreamingText('hello'));
    const next = buildLiveRunViewerOutputSignature('card-1', outputWithStreamingText('hello world'));

    expect(next).not.toBe(initial);
  });
});

function outputWithStreamingText(content: string): TaskRunCardOutput {
  return {
    committedMessages: [],
    streamingRows: [{
      key: 'stream-segment:assistant:1',
      message: buildAssistantMessage(content, 'stream-segment:assistant:1'),
    }],
  };
}
