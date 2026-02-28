import { deleteSession, getConfig, getSession, listSessions, sendHumanResponse, sendMessage, updateConfig } from './api';
import type { BridgeConfig, ConfigUpdate, SessionDetail, SessionMetadata } from './types';

describe('lib/api', () => {
  const fetchMock = jest.fn();

  beforeEach(() => {
    fetchMock.mockReset();
    Object.defineProperty(global, 'fetch', {
      value: fetchMock,
      writable: true,
    });
  });

  it('sendMessage posts to /api/agent and returns assistant response with session id', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        status: 'success',
        payload: { message: 'hello from bridge', session_id: 'session-123' },
        error: '',
      }),
    });

    const response = await sendMessage('hello');

    expect(response).toEqual({ message: 'hello from bridge', session_id: 'session-123' });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/agent',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ message: 'hello' }),
      })
    );
  });

  it('sendMessage includes session_id when provided', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        status: 'success',
        payload: { message: 'follow-up', session_id: 'session-abc' },
        error: '',
      }),
    });

    await sendMessage('hello again', 'session-abc');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/agent',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ message: 'hello again', session_id: 'session-abc' }),
      })
    );
  });

  it('sendHumanResponse posts HUMAN_RESPONSE action to /api/bus', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        status: 'success',
        payload: { accepted: true },
        error: '',
      }),
    });

    await sendHumanResponse(' session-1 ', ' q-1 ', '  PostgreSQL  ');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/bus',
      expect.objectContaining({
        method: 'POST',
        body: expect.any(String),
      })
    );

    const body = JSON.parse(fetchMock.mock.calls[0][1].body as string);
    expect(body).toEqual({
      action: 'HUMAN_RESPONSE',
      params: {
        session_id: 'session-1',
        question_id: 'q-1',
        answer: 'PostgreSQL',
      },
      trace_id: expect.stringMatching(/^web-\d+$/),
    });
  });

  it('getConfig reads /api/config with GET', async () => {
    const expected: BridgeConfig = {
      provider: 'openai',
      base_url: 'https://api.openai.com/v1',
      model: 'gpt-4o',
      chat_path: '',
      api_key_set: true,
    };

    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        status: 'success',
        payload: expected,
        error: '',
      }),
    });

    const config = await getConfig();

    expect(config).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/config', expect.any(Object));
  });

  it('updateConfig posts payload and returns updated config', async () => {
    const update: ConfigUpdate = {
      provider: 'custom',
      model: 'local-model',
      base_url: 'http://localhost:1234',
      api_key: 'secret',
      chat_path: '/v1/chat',
    };

    const expected: BridgeConfig = {
      provider: 'custom',
      base_url: 'http://localhost:1234',
      model: 'local-model',
      chat_path: '/v1/chat',
      api_key_set: true,
    };

    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        status: 'success',
        payload: expected,
        error: '',
      }),
    });

    const config = await updateConfig(update);

    expect(config).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/config',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(update),
      })
    );
  });

  it('listSessions reads /api/sessions with GET', async () => {
    const expected: SessionMetadata[] = [
      {
        id: 'session-1',
        created_at: '2026-02-28T10:00:00Z',
        updated_at: '2026-02-28T10:05:00Z',
        message_count: 2,
        token_count: 128,
      },
    ];

    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        status: 'success',
        payload: expected,
        error: '',
      }),
    });

    const sessions = await listSessions();
    expect(sessions).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/sessions', expect.any(Object));
  });

  it('getSession reads session detail by id', async () => {
    const expected: SessionDetail = {
      id: 'session-1',
      created_at: '2026-02-28T10:00:00Z',
      updated_at: '2026-02-28T10:05:00Z',
      token_count: 128,
      messages: [
        { role: 'user', text: 'hello' },
        { role: 'assistant', text: 'hi' },
      ],
    };

    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        status: 'success',
        payload: expected,
        error: '',
      }),
    });

    const session = await getSession('session-1');
    expect(session).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/sessions/session-1', expect.any(Object));
  });

  it('deleteSession sends delete request by id', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        status: 'success',
        payload: { id: 'session-1', deleted: true },
        error: '',
      }),
    });

    await deleteSession('session-1');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/sessions/session-1',
      expect.objectContaining({
        method: 'DELETE',
      })
    );
  });

  it('throws envelope error message for error responses', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({
        status: 'error',
        payload: {},
        error: 'bad request',
      }),
    });

    await expect(sendMessage('hello')).rejects.toThrow('bad request');
  });

  it('throws fallback status message when error field is empty', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 502,
      json: async () => ({
        status: 'error',
        payload: {},
        error: '',
      }),
    });

    await expect(getConfig()).rejects.toThrow('Request failed with status 502');
  });
});
