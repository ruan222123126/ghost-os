import { forwardBridge } from './forward';
import { createParamBridgeRouteHandler } from './route';

jest.mock('./forward', () => ({
  forwardBridge: jest.fn(),
}));

describe('lib/server/bridge/route', () => {
  beforeEach(() => {
    jest.mocked(forwardBridge).mockReset();
    jest.mocked(forwardBridge).mockResolvedValue(new Response(null, { status: 204 }));
  });

  it('forwards the incoming query string to the bridge path', async () => {
    const handler = createParamBridgeRouteHandler(
      'GET',
      ({ id }: { id: string }) => `/api/sessions/${id}`,
    );
    const request = new Request('http://localhost/api/sessions/session-1?limit=50&before=120');

    await handler(request, {
      params: {
        id: 'session-1',
      },
    });

    expect(forwardBridge).toHaveBeenCalledWith({
      path: '/api/sessions/session-1?limit=50&before=120',
      method: 'GET',
      request,
      headers: undefined,
    });
  });
});
