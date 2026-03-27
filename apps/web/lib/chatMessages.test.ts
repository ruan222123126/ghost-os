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
      { role: 'internal', text: '[GRAPHQL_EXECUTION_RESULT]\n{"data":{"viewer":{"id":"1"}}}' },
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

    expect(mapped).toHaveLength(3);
    expect(mapped[0]).toMatchObject({ kind: 'system', content: 'keep sharp' });
    expect(mapped[1]).toMatchObject({
      kind: 'system',
      content: '[GRAPHQL_EXECUTION_RESULT]\n{"data":{"viewer":{"id":"1"}}}',
    });
    expect(mapped[2]).toMatchObject({
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

  it('maps user image content into chat messages', () => {
    const messages: SessionMessage[] = [
      {
        role: 'user',
        text: '',
        content: [
          {
            type: 'image',
            image: {
              url: 'data:image/png;base64,R2hvc3Q=',
              mime_type: 'image/png',
              bytes: 5,
            },
          },
        ],
      },
    ];

    const mapped = mapSessionMessagesToChat(messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'user',
      content: '',
      images: [
        {
          url: 'data:image/png;base64,R2hvc3Q=',
          mimeType: 'image/png',
          bytes: 5,
        },
      ],
    });
  });

  it('keeps path-only user images visible instead of dropping them', () => {
    const messages: SessionMessage[] = [
      {
        role: 'user',
        text: 'inspect this',
        content: [
          {
            type: 'image',
            image: {
              path: '/tmp/cat.png',
              mime_type: 'image/png',
            },
          },
        ],
      },
    ];

    const mapped = mapSessionMessagesToChat(messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'user',
      content: 'inspect this',
      images: [
        {
          path: '/tmp/cat.png',
          mimeType: 'image/png',
        },
      ],
    });
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

  it('hides assistant graphql tool text when the turn already has a tool card', () => {
    const messages: SessionMessage[] = [
      { role: 'user', text: '搜一下 AI 咨询行业动态' },
      { role: 'assistant', text: 'mutation { web_search(provider: tavily, query: "AI consulting latest trends 2025") }' },
      {
        role: 'tool',
        tool_call_id: 'call-1',
        tool_result: {
          status: 'success',
          tool: 'web_search',
          trace_id: 'trace-web-1',
          output: 'search result',
          error: '',
        },
      },
      { role: 'assistant', text: '我整理了几条近期趋势。' },
    ];

    const mapped = mapSessionMessagesToChat(messages);

    expect(mapped).toHaveLength(3);
    expect(mapped[0]).toMatchObject({ kind: 'user', content: '搜一下 AI 咨询行业动态' });
    expect(mapped[1]).toMatchObject({ kind: 'tool', toolName: 'web_search', toolCallId: 'call-1' });
    expect(mapped[2]).toMatchObject({ kind: 'assistant', content: '我整理了几条近期趋势。' });
  });

  it('keeps assistant graphql text visible when no tool card follows', () => {
    const messages: SessionMessage[] = [
      { role: 'assistant', text: 'mutation { web_search(provider: tavily, query: "OpenAI") }' },
    ];

    const mapped = mapSessionMessagesToChat(messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'assistant',
      content: 'mutation { web_search(provider: tavily, query: "OpenAI") }',
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
