import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';
import { getMousePosition } from './mousePosition';

describe('lib/api/tools/mousePosition', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('getMousePosition parses payload', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        x: 320,
        y: 640,
        display_id: 3,
        scale_x: 2,
        scale_y: 2,
      },
      error: '',
    });

    await expect(getMousePosition()).resolves.toEqual({
      x: 320,
      y: 640,
      display_id: 3,
      scale_x: 2,
      scale_y: 2,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/tools/screen/mouse-position',
      expect.objectContaining({
        method: 'POST',
      }),
    );
  });
});
