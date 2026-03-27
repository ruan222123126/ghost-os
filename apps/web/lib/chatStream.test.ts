import {
  appendAssistantText,
  replaceAssistantText,
  suppressGraphQLAssistantToolText,
  upsertPendingQuestion,
  upsertToolMessage,
} from './chatStream';
import type { ChatMessage } from './types';

describe('lib/chatStream', () => {
  it('appends assistant deltas into a single assistant message', () => {
    let messages: ChatMessage[] = [];

    messages = appendAssistantText(messages, {
      messageId: 'assistant-1',
      text: 'hello',
    });
    messages = appendAssistantText(messages, {
      messageId: 'assistant-1',
      text: ' world',
    });

    expect(messages).toEqual([
      { id: 'assistant-1', kind: 'assistant', content: 'hello world' },
    ]);
  });

  it('replaces the streamed assistant message with the final message payload', () => {
    const messages = replaceAssistantText([{
      id: 'assistant-1',
      kind: 'assistant',
      content: 'hello wor',
    }], {
      messageId: 'assistant-1',
      text: 'hello world',
    });

    expect(messages).toEqual([
      { id: 'assistant-1', kind: 'assistant', content: 'hello world' },
    ]);
  });

  it('removes streamed assistant graphql tool text once a tool card exists', () => {
    const messages = suppressGraphQLAssistantToolText([{
      id: 'assistant-1',
      kind: 'assistant',
      content: 'mutation { web_search(provider: tavily, query: "OpenAI") }',
    }], 'assistant-1');

    expect(messages).toEqual([]);
  });

  it('upserts tool messages by message id', () => {
    let messages: ChatMessage[] = [];

    messages = upsertToolMessage(messages, {
      messageId: 'tool-1',
      content: 'script_exec running',
      toolCallId: 'call-1',
      toolName: 'script_exec',
      toolStatus: 'running',
      traceId: 'trace-1',
    });
    messages = upsertToolMessage(messages, {
      messageId: 'tool-1',
      content: 'script_exec finished',
      toolStatus: 'success',
    });

    expect(messages).toEqual([{
      id: 'tool-1',
      kind: 'tool',
      content: 'script_exec finished',
      toolCallId: 'call-1',
      toolName: 'script_exec',
      toolStatus: 'success',
      traceId: 'trace-1',
    }]);
  });

  it('upserts pending question messages by message id', () => {
    let messages: ChatMessage[] = [];

    messages = upsertPendingQuestion(messages, {
      messageId: 'question-1',
      content: 'Choose a database',
      questionId: 'q-1',
      sessionId: 'session-1',
      selectionMode: 'single',
    });
    messages = upsertPendingQuestion(messages, {
      messageId: 'question-1',
      content: 'Choose a primary database',
      questionId: 'q-1',
      sessionId: 'session-1',
      selectionMode: 'single',
      options: [{ label: 'PostgreSQL' }],
    });

    expect(messages).toEqual([{
      id: 'question-1',
      kind: 'pending_question',
      content: 'Choose a primary database',
      questionId: 'q-1',
      sessionId: 'session-1',
      selectionMode: 'single',
      options: [{ label: 'PostgreSQL' }],
    }]);
  });
});
