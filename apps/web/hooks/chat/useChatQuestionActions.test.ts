import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { PendingQuestionMessage } from '@/lib/types';
import { useChatQuestionActions } from './useChatQuestionActions';

describe('hooks/chat/useChatQuestionActions', () => {
  it('answers pending questions through the shared human response runner', async () => {
    const options = buildOptions();
    const latest = renderQuestionActions(options);

    await act(async () => {
      await latest.current.answerQuestion('question-1', ' approved ');
    });

    expect(options.clearChatError).toHaveBeenCalledWith('session-1');
    expect(options.clearStreamingState).toHaveBeenCalledWith('session-1');
    expect(options.removePendingQuestion).toHaveBeenCalledWith('session-1', 'question-1');
    expect(options.appendCommittedMessages).toHaveBeenCalledWith(
      'session-1',
      [expect.objectContaining({ kind: 'user', content: 'approved' })],
    );
    expect(options.runHumanStream).toHaveBeenCalledWith(expect.objectContaining({
      answer: 'approved',
      questionId: 'question-1',
      sessionId: 'session-1',
    }));
    expect(options.markBackgroundCompleted).toHaveBeenCalledWith('session-1');
    expect(options.setLoading).toHaveBeenLastCalledWith('session-1', false);
    expect(options.setActiveRun).toHaveBeenLastCalledWith('session-1', null);
  });

  it('cancels pending questions without appending an answer message', async () => {
    const options = buildOptions();
    const latest = renderQuestionActions(options);

    await act(async () => {
      await latest.current.cancelQuestion('question-1');
    });

    expect(options.removePendingQuestion).toHaveBeenCalledWith('session-1', 'question-1');
    expect(options.appendCommittedMessages).not.toHaveBeenCalled();
    expect(options.runHumanStream).toHaveBeenCalledWith(expect.objectContaining({
      answer: '',
      cancelled: true,
      questionId: 'question-1',
      sessionId: 'session-1',
    }));
    expect(options.setLoading).toHaveBeenLastCalledWith('session-1', false);
  });

  it('rejects empty answers before starting a human response run', async () => {
    const options = buildOptions();
    const latest = renderQuestionActions(options);

    await act(async () => {
      await latest.current.answerQuestion('question-1', '   ');
    });

    expect(options.setChatError).toHaveBeenCalledWith('session-1', 'Answer cannot be empty');
    expect(options.runHumanStream).not.toHaveBeenCalled();
  });
});

function renderQuestionActions(options: QuestionActionOptions) {
  const latest: { current: ReturnType<typeof useChatQuestionActions> } = {
    current: null as unknown as ReturnType<typeof useChatQuestionActions>,
  };

  act(() => {
    TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: 'en-US',
        children: React.createElement(QuestionActionsProbe, {
          options,
          onRender: (state) => {
            latest.current = state;
          },
        }),
      }),
    );
  });

  return latest;
}

function QuestionActionsProbe(props: {
  options: QuestionActionOptions;
  onRender: (state: ReturnType<typeof useChatQuestionActions>) => void;
}) {
  const state = useChatQuestionActions(props.options);
  props.onRender(state);
  return null;
}

function buildOptions(overrides: Partial<QuestionActionOptions> = {}): QuestionActionOptions {
  return {
    appendCommittedMessages: jest.fn(),
    appendErrorMessage: jest.fn(),
    clearChatError: jest.fn(),
    clearStreamingState: jest.fn(),
    pendingQuestions: [buildQuestion()],
    getCurrentSessionId: jest.fn(() => 'current-session'),
    getStopPending: jest.fn(() => false),
    hasPendingQuestionInSession: jest.fn(() => false),
    markBackgroundCompleted: jest.fn(),
    runHumanStream: jest.fn().mockResolvedValue({ sessionId: 'session-1', terminalType: 'done' }),
    removePendingQuestion: jest.fn(),
    resolveActiveRunSessionId: jest.fn((_traceId, fallbackSessionId) => fallbackSessionId),
    setActiveRun: jest.fn(),
    setChatError: jest.fn(),
    setLoading: jest.fn(),
    setStopPending: jest.fn(),
    ...overrides,
  };
}

function buildQuestion(overrides: Partial<PendingQuestionMessage> = {}): PendingQuestionMessage {
  return {
    id: 'pending-1',
    kind: 'pending_question',
    content: 'Approve?',
    questionId: 'question-1',
    sessionId: 'session-1',
    ...overrides,
  };
}

type QuestionActionOptions = Parameters<typeof useChatQuestionActions>[0];
