import { isAgentRunCancellationMessage } from './errors';

describe('errors', () => {
  it('detects wrapped agent run cancellation messages', () => {
    expect(isAgentRunCancellationMessage('agent run cancelled')).toBe(true);
    expect(isAgentRunCancellationMessage('agent stream closed before terminal event')).toBe(true);
    expect(isAgentRunCancellationMessage('context canceled')).toBe(true);
    expect(isAgentRunCancellationMessage(
      'trace_id=agent-run-f35b5652-da4f-4f78-833c-a77154adba62 turn=7 complete_once: stream interrupted: context canceled',
    )).toBe(true);
  });

  it('does not classify unrelated stream failures as cancellations', () => {
    expect(isAgentRunCancellationMessage('stream interrupted: context deadline exceeded')).toBe(false);
    expect(isAgentRunCancellationMessage('complete_once: provider returned 500')).toBe(false);
  });
});
