import { buildRSSBriefing, getRSSBriefing } from './api';
import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';
import type { RSSBriefing } from '@/lib/types';

describe('lib/api/rss/api', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('getRSSBriefing returns null for 404 responses', async () => {
    mockFetchJSON(
      {
        status: 'error',
        payload: {},
        error: 'not found',
      },
      { ok: false, status: 404 },
    );

    await expect(getRSSBriefing()).resolves.toBeNull();
  });

  it('buildRSSBriefing posts payload and returns parsed briefing', async () => {
    const expected: RSSBriefing = {
      id: 'briefing-1',
      title: 'Daily RSS Briefing',
      generated_at: '2026-03-14T00:00:00Z',
      window_hours: 24,
      scanned_groups: 4,
      highlight_count: 1,
      highlights: [
        {
          rank: 1,
          group_id: 'group-1',
          headline: 'Bridge API refactor shipped',
          importance: 'high',
          source_item_count: 3,
          source_feed_count: 2,
        },
      ],
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const response = await buildRSSBriefing({ window_hours: 24, highlights_limit: 5 });

    expect(response).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/rss/briefing',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ window_hours: 24, highlights_limit: 5 }),
      }),
    );
  });
});
