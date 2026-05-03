import { mapSessionMessagesToChat } from './chatMessages';
import type { SessionMessage } from './types';

describe('chatMessages bash_exec hydration', () => {
  it('hydrates tool messages with matching assistant tool_calls by tool_call_id', () => {
    const messages: SessionMessage[] = [
      {
        index: 0,
        role: 'assistant',
        text: '',
        tool_calls: [
          {
            id: 'call-bash-1',
            name: 'bash_exec',
            arguments: { command: 'which agent-browser' },
          },
        ],
      },
      {
        index: 1,
        role: 'tool',
        text: '/home/ruan/.nvm/versions/node/v24.14.0/bin/agent-browser',
        tool_call_id: 'call-bash-1',
        tool_result: {
          status: 'success',
          tool: 'bash_exec',
          trace_id: 'trace-bash-1',
          output: '/home/ruan/.nvm/versions/node/v24.14.0/bin/agent-browser',
          error: '',
        },
      },
    ];

    const mapped = mapSessionMessagesToChat('session-bash', messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'tool',
      toolName: 'bash_exec',
      toolCallId: 'call-bash-1',
      toolCalls: [
        {
          id: 'call-bash-1',
          name: 'bash_exec',
          arguments: { command: 'which agent-browser' },
        },
      ],
    });
  });
});
