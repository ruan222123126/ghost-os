import {
  parseAgentCompletionDeltaPayload,
  parseAgentErrorPayload,
  parseAgentSendResponse,
  parseAgentStopResponse,
  parseAgentToolCallFinishedPayload,
  parseAgentToolCallStartedPayload,
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

  it('rejects removed prox success mode', () => {
    expect(() => parseAgentSendResponse({
      message: 'legacy response',
      session_id: 'session-legacy',
      session_ended: false,
      mode: 'prox',
    })).toThrow('agent response.mode');
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

  it('parses stop payloads with optional session id', () => {
    expect(parseAgentStopResponse({
      status: 'stopped',
      message: 'agent run cancelled successfully',
      session_id: 'session-stop',
      ignored: true,
    })).toEqual({
      status: 'stopped',
      message: 'agent run cancelled successfully',
      session_id: 'session-stop',
    });
  });

  it('parses thinking completion delta payloads', () => {
    expect(parseAgentCompletionDeltaPayload({
      kind: 'thinking',
      thinking: 'Analyzing…',
      ignored: true,
    })).toEqual({
      kind: 'thinking',
      thinking: 'Analyzing…',
      text: undefined,
      tool_call_index: undefined,
      tool_call_id: undefined,
      tool_name: undefined,
      arguments_fragment: undefined,
    });
  });

  it('parses tool lifecycle payload fields for arguments and output', () => {
    expect(parseAgentToolCallStartedPayload({
      tool: 'script_exec',
      tool_call_id: 'call-1',
      arguments_json: '{"script":"print(1)"}',
    })).toEqual({
      tool: 'script_exec',
      tool_call_id: 'call-1',
      arguments_json: '{"script":"print(1)"}',
    });

    expect(parseAgentToolCallFinishedPayload({
      tool: 'script_exec',
      tool_call_id: 'call-1',
      status: 'success',
      output: '{"summary":{"step_count":1},"steps":[]}',
    })).toEqual({
      tool: 'script_exec',
      tool_call_id: 'call-1',
      status: 'success',
      error: undefined,
      output: '{"summary":{"step_count":1},"steps":[]}',
    });
  });
});
