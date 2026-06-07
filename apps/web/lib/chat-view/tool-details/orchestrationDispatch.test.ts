import type { ToolChatMessage } from '@/lib/types';
import { formatToolDetails } from './format';

describe('orchestration dispatch compact details', () => {
  it('renders public_once member speeches in compact mode', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'orchestration_dispatch',
      toolStatus: 'success',
      rawOutput: JSON.stringify({
        action: 'public_once',
        participant_ids: ['agent-2'],
        member_results: [
          { agent_id: 'agent-2', title: '玩家1', content: '我是好人，这轮先听法官安排。' },
        ],
      }),
    }), { compactOutputEnabled: true });

    expect(details).toBe('玩家1：我是好人，这轮先听法官安排。');
  });

  it('renders private_once participants plus transcript in compact mode', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'orchestration_dispatch',
      toolStatus: 'success',
      rawOutput: JSON.stringify({
        action: 'private_once',
        participant_ids: ['agent-1', 'agent-2'],
        private_transcript: [
          { speaker: 'system', content: 'shared context' },
          { speaker: '法官', agent_id: 'agent-1', content: '今晚你要查验谁？' },
          { speaker: '玩家1', agent_id: 'agent-2', content: '我查验 3 号。' },
        ],
      }),
    }), { compactOutputEnabled: true });

    expect(details).toBe('私密子回合\n法官：今晚你要查验谁？\n玩家1：我查验 3 号。');
  });

  it('renders public_once args before result hydration in compact mode', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'orchestration_dispatch',
      toolStatus: 'running',
      content: JSON.stringify({
        action: 'public_once',
        participant_ids: ['agent-2', 'agent-3'],
        instruction: '按顺序公开发言',
      }),
    }), { compactOutputEnabled: true });

    expect(details).toBe('公开轮\n指令：按顺序公开发言');
  });
});

function buildToolMessage(overrides: Partial<ToolChatMessage>): ToolChatMessage {
  return {
    id: 'tool-1',
    kind: 'tool',
    content: '',
    ...overrides,
  };
}
