import { sendHumanResponse, sendMessage, stopAgent } from './api';
import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';

describe('lib/api/agent/api', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('sendMessage posts to /api/agent and returns assistant response with session id', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { message: 'hello from bridge', session_id: 'session-123', session_ended: false },
      error: '',
    });

    const response = await sendMessage('hello');

    expect(response).toEqual({ message: 'hello from bridge', session_id: 'session-123', session_ended: false });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/agent',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ message: 'hello' }),
      }),
    );
  });

  it('sendMessage includes session_id and trace_id when provided', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { message: 'follow-up', session_id: 'session-abc', session_ended: false },
      error: '',
    });

    await sendMessage('hello again', undefined, 'session-abc', 'trace-abc');

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(init.method).toBe('POST');
    expect(JSON.parse(String(init.body))).toEqual({
      message: 'hello again',
      session_id: 'session-abc',
      trace_id: 'trace-abc',
    });
  });

  it('sendMessage includes images when provided', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { message: 'done', session_id: 'session-img', session_ended: false },
      error: '',
    });

    await sendMessage('', [{
      url: 'data:image/png;base64,R2hvc3Q=',
      mime_type: 'image/png',
      bytes: 5,
    }], 'session-img', 'trace-img');

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(JSON.parse(String(init.body))).toEqual({
      message: '',
      images: [{
        url: 'data:image/png;base64,R2hvc3Q=',
        mime_type: 'image/png',
        bytes: 5,
      }],
      session_id: 'session-img',
      trace_id: 'trace-img',
    });
  });

  it('stopAgent posts AGENT_STOP to /api/bus', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { status: 'stopped', message: 'agent run cancelled successfully' },
      error: '',
    });

    const response = await stopAgent(' session-1 ', ' trace-run ');

    expect(response).toEqual({ status: 'stopped', message: 'agent run cancelled successfully' });
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(fetchMock).toHaveBeenCalledWith('/api/bus', expect.objectContaining({ method: 'POST' }));
    expect(JSON.parse(String(init.body))).toEqual({
      action: 'AGENT_STOP',
      params: {
        session_id: 'session-1',
        trace_id: 'trace-run',
      },
      trace_id: expect.stringMatching(/^agent-stop-/),
    });
  });

  it('stopAgent requires session_id or trace_id', async () => {
    await expect(stopAgent()).rejects.toThrow('session_id or trace_id is required');
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('sendHumanResponse posts answer payload to /api/questions/answer and returns agent response', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { message: 'PostgreSQL sounds good', session_id: 'session-1', session_ended: false },
      error: '',
    });

    const response = await sendHumanResponse(' session-1 ', ' q-1 ', '  PostgreSQL  ');

    expect(response).toEqual({ message: 'PostgreSQL sounds good', session_id: 'session-1', session_ended: false });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/questions/answer',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          session_id: 'session-1',
          question_id: 'q-1',
          answer: 'PostgreSQL',
        }),
      }),
    );
  });

  it('sendHumanResponse can cancel a pending question', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        message: 'Conversation cancelled by user.',
        session_id: 'session-1',
        session_ended: true,
        session_end: { signal: 'END_SESSION', message: 'Conversation cancelled by user.' },
      },
      error: '',
    });

    const response = await sendHumanResponse(' session-1 ', ' q-1 ', '  ', true);

    expect(response).toEqual({
      message: 'Conversation cancelled by user.',
      session_id: 'session-1',
      session_ended: true,
      session_end: { signal: 'END_SESSION', message: 'Conversation cancelled by user.' },
    });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/questions/answer',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          session_id: 'session-1',
          question_id: 'q-1',
          answer: '',
          cancelled: true,
        }),
      }),
    );
  });

  it('throws envelope error message for error responses', async () => {
    mockFetchJSON(
      {
        status: 'error',
        payload: {},
        error: 'bad request',
      },
      { ok: false, status: 400 },
    );

    await expect(sendMessage('hello')).rejects.toThrow('bad request');
  });
});
