import { stopAgent } from './api';
import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';

describe('lib/api/agent/api', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('stopAgent posts AGENT_STOP to /api/bus', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        status: 'stopped',
        message: 'agent run cancelled successfully',
        session_id: 'session-1',
      },
      error: '',
    });

    const response = await stopAgent(' session-1 ', ' trace-run ');

    expect(response).toEqual({
      status: 'stopped',
      message: 'agent run cancelled successfully',
      session_id: 'session-1',
    });
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

  it('throws envelope error message for error responses', async () => {
    mockFetchJSON(
      {
        status: 'error',
        payload: {},
        error: 'bad request',
      },
      { ok: false, status: 400 },
    );

    await expect(stopAgent('session-1')).rejects.toThrow('bad request');
  });
});
