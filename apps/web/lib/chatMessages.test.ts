import {
  buildPendingQuestionMessage,
  hasPendingQuestion,
  mapAgentReplyToChatMessages,
  mapSessionMessagesToChat,
  replacePendingQuestionWithUserAnswer,
} from './chatMessages';
import type { AgentSendAwaitingHumanResponse, SessionMessage } from './types';

describe('chatMessages', () => {
  it('preserves system and tool messages as structured chat kinds', () => {
    const messages: SessionMessage[] = [
      { role: 'system', text: 'keep sharp' },
      {
        role: 'tool',
        text: 'README.md contents',
        tool_result: {
          status: 'success',
          tool: 'read_file',
          trace_id: 'trace-1',
          output: 'README.md contents',
          error: '',
        },
        tool_call_id: 'call-1',
      },
    ];

    const mapped = mapSessionMessagesToChat(messages);

    expect(mapped).toHaveLength(2);
    expect(mapped[0]).toMatchObject({ kind: 'system', content: 'keep sharp' });
    expect(mapped[1]).toMatchObject({
      kind: 'tool',
      content: 'README.md contents',
      toolName: 'read_file',
      toolCallId: 'call-1',
      traceId: 'trace-1',
    });
  });

  it('reconstructs answered ask_human tool results into question and user timeline', () => {
    const messages: SessionMessage[] = [
      {
        role: 'tool',
        text: 'Which database should I use?\nPostgreSQL',
        tool_result: {
          status: 'success',
          tool: 'ask_human',
          trace_id: 'trace-2',
          error: '',
        },
        human_interaction: {
          question_id: 'q-1',
          prompt: 'Which database should I use?',
          selection_mode: 'single',
          options: [
            { label: 'PostgreSQL' },
            { label: 'Other', allow_custom: true },
          ],
          answer: 'PostgreSQL',
        },
      },
    ];

    const mapped = mapSessionMessagesToChat(messages);

    expect(mapped).toHaveLength(2);
    expect(mapped[0]).toMatchObject({
      kind: 'question',
      content: 'Which database should I use?',
      questionId: 'q-1',
      selectionMode: 'single',
      options: [
        { label: 'PostgreSQL' },
        { label: 'Other', allow_custom: true },
      ],
    });
    expect(mapped[1]).toMatchObject({ kind: 'user', content: 'PostgreSQL' });
  });

  it('treats tool text as plain content when structured tool_result is absent', () => {
    const rawEnvelope = JSON.stringify({
      status: 'success',
      tool: 'read_file',
      trace_id: 'trace-legacy',
      output: 'README.md contents',
      error: '',
    });
    const messages: SessionMessage[] = [{ role: 'tool', text: rawEnvelope }];

    const mapped = mapSessionMessagesToChat(messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'tool',
      content: rawEnvelope,
      toolName: undefined,
      traceId: undefined,
    });
  });

  it('replaces pending question cards with the user answer', () => {
    const pendingReply: AgentSendAwaitingHumanResponse = {
      status: 'awaiting_human',
      session_id: 'session-1',
      question_id: 'q-1',
      prompt: 'Continue deploy?',
      selection_mode: 'multiple',
      options: [
        { label: 'Ship now' },
        { label: 'Wait for QA' },
        { label: 'Other', allow_custom: true },
      ],
    };
    const pending = buildPendingQuestionMessage(pendingReply);

    expect(hasPendingQuestion([pending])).toBe(true);
    expect(mapAgentReplyToChatMessages(pendingReply)[0]).toMatchObject({
      kind: 'pending_question',
      questionId: 'q-1',
      selectionMode: 'multiple',
      options: [
        { label: 'Ship now' },
        { label: 'Wait for QA' },
        { label: 'Other', allow_custom: true },
      ],
    });

    const replaced = replacePendingQuestionWithUserAnswer([pending], 'q-1', 'Yes');

    expect(replaced).toHaveLength(1);
    expect(replaced[0]).toMatchObject({ kind: 'user', content: 'Yes' });
    expect(hasPendingQuestion(replaced)).toBe(false);
  });
});
