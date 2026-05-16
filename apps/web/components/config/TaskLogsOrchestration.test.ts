import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { OrchestrationRoundsBlock, parseOrchestrationGroupOutput } from './TaskLogsOrchestration';

describe('components/config/TaskLogsOrchestration', () => {
  it('renders full owner dispatch details for user-visible logs', () => {
    const output = parseOrchestrationGroupOutput({
      completed_rounds: 1,
      owner_agent_id: 'agent-1',
      owner_session_id: 'owner-session',
      member_results: [
        { round: 1, title: 'Member', status: 'success', content: 'done' },
      ],
      dispatch_results: [
        {
          round: 1,
          action: 'private_send',
          owner_visible: false,
          private_deliveries: [
            { participant_id: 'agent-2', content: 'secret-role' },
          ],
          private_transcript: [
            { round: 1, speaker: 'Owner', content: 'keep all details' },
          ],
        },
      ],
    });

    expect(output?.dispatch_results?.[0].private_deliveries?.[0].content).toBe('secret-role');

    const html = renderToStaticMarkup(
      React.createElement(OrchestrationRoundsBlock, {
        output: output as NonNullable<typeof output>,
      }),
    );
    expect(html).toContain('dispatch 1: private_send');
    expect(html).toContain('private_send: agent-2 &lt;- secret-role');
    expect(html).toContain('owner_visible: false');
    expect(html).toContain('private transcript');
    expect(html).toContain('round 1 Owner: keep all details');
    expect(html).toContain('Member [success]: done');
  });
});
