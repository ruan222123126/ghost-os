import {
  buildFindIconTemplatePreviewURL,
  previewFindIcon,
  uploadFindIconTemplate,
} from './findIcon';
import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';

describe('lib/api/tools/findIcon', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('uploadFindIconTemplate calls template upload endpoint', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        template_path: '/tmp/abcd.png',
        template_name: 'abcd.png',
        sha256: 'a'.repeat(64),
      },
      error: '',
    });

    await uploadFindIconTemplate({
      filename: 'button.png',
      mime_type: 'image/png',
      data_url: 'data:image/png;base64,R2hvc3Q=',
    });

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/tools/screen/find-icon/template',
      expect.objectContaining({
        method: 'POST',
      }),
    );
  });

  it('previewFindIcon parses preview payload', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        exists: true,
        match_count: 1,
        matches: [{ score: 0.98 }],
        display_id: 2,
        region: { x: 1, y: 2, width: 300, height: 200 },
        hovered: true,
      },
      error: '',
    });

    await expect(previewFindIcon({
      template_path: '/tmp/icon.png',
      threshold: 0.9,
    })).resolves.toEqual({
      exists: true,
      match_count: 1,
      matches: [{ score: 0.98 }],
      display_id: 2,
      region: { x: 1, y: 2, width: 300, height: 200 },
      hovered: true,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/tools/screen/find-icon/preview',
      expect.objectContaining({
        method: 'POST',
      }),
    );
  });

  it('buildFindIconTemplatePreviewURL encodes template path in query', () => {
    expect(buildFindIconTemplatePreviewURL('/tmp/icon a.png'))
      .toBe('/api/tools/screen/find-icon/template-preview?template_path=%2Ftmp%2Ficon%20a.png');
  });
});
