import {
  buildPendingQuestionMessage,
  hasPendingQuestion,
  mapAgentReplyToChatMessages,
  mapSessionMessagesToChat,
  replacePendingQuestionWithUserAnswer,
} from './chatMessages';
import { formatToolDetails } from './chat-view/tool-details/format';
import { buildAgentMessageWithSelectedSkill } from './selectedSkillMessage';
import type { AgentSendAwaitingHumanResponse, SessionMessage } from './types';

const SESSION_ID = 'session-test';

describe('chatMessages', () => {
  it('preserves system and tool messages as structured chat kinds', () => {
    const messages = withSessionIndices([
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
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(3);
    expect(mapped[0]).toMatchObject({ kind: 'system', content: 'keep sharp', sourceRole: 'system' });
    expect(mapped[1]).toMatchObject({
      kind: 'system',
      content: '[GRAPHQL_EXECUTION_RESULT]\n{"data":{"viewer":{"id":"1"}}}',
      sourceRole: 'internal',
    });
    expect(mapped[2]).toMatchObject({
      kind: 'tool',
      content: 'README.md contents',
      toolName: 'read_file',
      toolCallId: 'call-1',
      traceId: 'trace-1',
    });
  });

  it('filters tool-tag sfind internal notes to loaded items only', () => {
    const messages = withSessionIndices([
      {
        role: 'internal',
        text: '[TOOL_TAG_RESULT]\n{"tool":"sfind","output":{"action":"list","items":[{"name":"release_flow","status":"active","available_now":true},{"name":"incident_triage","status":"expired"},{"name":"ship_checklist","status":"pending","available_next_turn":true}]}}',
      },
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({ kind: 'system', sourceRole: 'internal' });
    expect(mapped[0].content).toBe(
      '[TOOL_TAG_RESULT]\n{"tool":"sfind","output":{"action":"list","items":[{"name":"release_flow","status":"active","available_now":true},{"name":"ship_checklist","status":"pending","available_next_turn":true}]}}',
    );
  });

  it('restores selected skill metadata from wrapped user history messages', () => {
    const messages = withSessionIndices([
      {
        role: 'user',
        text: buildAgentMessageWithSelectedSkill({
          images: [],
          message: 'ship release',
          selectedSkill: {
            id: 'skill_release',
            name: 'release_flow',
          },
        }),
      },
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'user',
      content: 'ship release',
      selectedSkill: {
        id: 'skill_release',
        name: 'release_flow',
      },
    });
  });

  it('maps task run event internal notes to divider events', () => {
    const messages = withSessionIndices([
      {
        role: 'internal',
        text: '[TASK_RUN_EVENT]\n工作流 if 判断：if-node branch=true -> agent-true',
      },
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'event',
      content: '工作流 if 判断：if-node branch=true -> agent-true',
    });
  });

  it('reconstructs answered ask_human tool results into question and user timeline', () => {
    const messages = withSessionIndices([
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
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

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
    const messages = withSessionIndices([
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
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

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

  it('maps tool image content into tool chat message images', () => {
    const messages = withSessionIndices([
      {
        role: 'tool',
        text: 'Generated 1 image(s).',
        content: [
          {
            type: 'image',
            image: {
              url: '/api/sessions/session-test/artifacts/artifact-1',
              mime_type: 'image/png',
              width: 512,
              height: 512,
              bytes: 1024,
              sha256: 'abc',
            },
          },
        ],
        tool_result: {
          status: 'success',
          tool: 'screen_action',
          output: 'Generated 1 image(s).',
        },
      },
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'tool',
      toolName: 'screen_action',
      images: [
        {
          url: '/api/sessions/session-test/artifacts/artifact-1',
          mimeType: 'image/png',
          width: 512,
          height: 512,
          bytes: 1024,
          sha256: 'abc',
        },
      ],
    });
  });

  it('keeps path-only user images visible instead of dropping them', () => {
    const messages = withSessionIndices([
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
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

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
    const messages = withSessionIndices([{ role: 'tool', text: rawEnvelope }]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'tool',
      content: rawEnvelope,
      toolName: undefined,
      traceId: undefined,
    });
  });

  it('still formats legacy orchestration_dispatch tool envelopes in compact mode', () => {
    const rawEnvelope = JSON.stringify({
      status: 'success',
      tool: 'orchestration_dispatch',
      trace_id: 'trace-owner-legacy',
      output: JSON.stringify({
        action: 'private_send',
        private_deliveries: [
          { participant_id: 'agent-2', content: 'legacy secret' },
          { participant_id: 'agent-3', content: 'legacy follow-up' },
        ],
      }),
      error: '',
    });
    const messages = withSessionIndices([{ role: 'tool', text: rawEnvelope }]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'tool',
      toolName: undefined,
      content: rawEnvelope,
    });
    expect(formatToolDetails(mapped[0] as Extract<typeof mapped[number], { kind: 'tool' }>, { compactOutputEnabled: true }))
      .toBe('向agent-2发了私信：legacy secret\n向agent-3发了私信：legacy follow-up');
  });

  it('hides assistant tool-tag text when the turn already has a tool card', () => {
    const messages = withSessionIndices([
      { role: 'user', text: '搜一下 AI 咨询行业动态' },
      { role: 'assistant', text: '<t:1>{"provider":"tavily","query":"AI consulting latest trends 2025"}</t>' },
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
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(3);
    expect(mapped[0]).toMatchObject({ kind: 'user', content: '搜一下 AI 咨询行业动态' });
    expect(mapped[1]).toMatchObject({ kind: 'tool', toolName: 'web_search', toolCallId: 'call-1' });
    expect(mapped[2]).toMatchObject({ kind: 'assistant', content: '我整理了几条近期趋势。' });
  });

  it('strips tool tags from assistant text and keeps visible prose', () => {
    const messages = withSessionIndices([
      { role: 'assistant', text: '我先查一下<t:1>{"provider":"tavily","query":"OpenAI"}</t>完成后给你总结。' },
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'assistant',
      content: '我先查一下完成后给你总结。',
    });
  });

  it('renders thinking before assistant text from session history', () => {
    const messages = withSessionIndices([
      { role: 'assistant', text: '结论如下。', thinking: '先分析上下文' },
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(2);
    expect(mapped[0]).toMatchObject({ kind: 'thinking', content: '先分析上下文' });
    expect(mapped[1]).toMatchObject({ kind: 'assistant', content: '结论如下。' });
  });

  it('keeps thinking-only tool loop assistants in history order', () => {
    const messages = withSessionIndices([
      { role: 'assistant', thinking: '先准备调用工具', tool_calls: [{ id: 'call-1', name: 'read_file', arguments: { path: 'README.md' } }] },
      {
        role: 'tool',
        text: 'README.md contents',
        tool_call_id: 'call-1',
        tool_result: {
          status: 'success',
          tool: 'read_file',
          output: 'README.md contents',
        },
      },
      { role: 'assistant', thinking: '工具结果已返回', text: '最终答案' },
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped.map((message) => message.kind)).toEqual([
      'thinking',
      'tool',
      'thinking',
      'assistant',
    ]);
  });

  it('does not render pure tool-tag assistant messages as raw text', () => {
    const messages = withSessionIndices([
      { role: 'assistant', text: '<t:1>{"provider":"tavily","query":"OpenAI"}</t>' },
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(0);
  });

  it('marks assistant draft messages as in progress', () => {
    const messages = withSessionIndices([
      { role: 'assistant', text: 'partial answer', in_progress: true },
    ]);

    const mapped = mapSessionMessagesToChat(SESSION_ID, messages);

    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toMatchObject({
      kind: 'assistant',
      content: 'partial answer',
      inProgress: true,
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

function withSessionIndices(messages: Omit<SessionMessage, 'index'>[]): SessionMessage[] {
  return messages.map((message, index) => ({
    index,
    ...message,
  }));
}
