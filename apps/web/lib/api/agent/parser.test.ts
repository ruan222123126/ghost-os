import {
  parseAgentErrorPayload,
  parseAgentSendResponse,
} from './parser';

describe('lib/api/agent/parser', () => {
  it('parses success payloads with plan mode and ignores unknown fields', () => {
    const payload = {
      message: 'plan only response',
      session_id: 'session-plan',
      session_ended: false,
      mode: 'plan',
      future_field: 'ignored',
    };

    expect(parseAgentSendResponse(payload)).toEqual({
      message: 'plan only response',
      session_id: 'session-plan',
      session_ended: false,
      mode: 'plan',
    });
  });

  it('parses awaiting human responses', () => {
    const payload = {
      status: 'awaiting_human',
      session_id: 'session-1',
      question_id: 'q-1',
      prompt: 'Continue?',
      selection_mode: 'single',
      options: [{ label: 'Continue' }],
      extra: true,
    };

    expect(parseAgentSendResponse(payload)).toEqual({
      status: 'awaiting_human',
      session_id: 'session-1',
      question_id: 'q-1',
      prompt: 'Continue?',
      selection_mode: 'single',
      options: [{ label: 'Continue' }],
    });
  });

  it('parses error payload code when present', () => {
    const payload = {
      message: 'bridge unavailable',
      session_id: 'session-err',
      code: 502,
      ignored: 'field',
    };

    expect(parseAgentErrorPayload(payload)).toEqual({
      message: 'bridge unavailable',
      session_id: 'session-err',
      code: 502,
    });
  });
});
