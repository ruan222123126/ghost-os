import { parseRSSBriefing } from './parser';
import type { RSSBriefing } from '@/lib/types';

describe('lib/api/rss/parser', () => {
  it('parses rss briefing payloads', () => {
    const payload: RSSBriefing = {
      id: 'briefing-1',
      title: 'Daily RSS Briefing',
      generated_at: '2026-03-14T00:00:00Z',
      saved_at: '2026-03-14T00:05:00Z',
      window_hours: 24,
      scanned_groups: 4,
      highlight_count: 1,
      trace_id: 'trace-1',
      task_id: 'task-1',
      highlights: [
        {
          rank: 1,
          group_id: 'group-1',
          headline: 'Bridge API refactor shipped',
          importance: 'high',
          source_item_count: 3,
          source_feed_count: 2,
          tags: ['bridge'],
        },
      ],
    };

    expect(parseRSSBriefing(payload)).toEqual(payload);
  });

  it('rejects unexpected highlight importance values', () => {
    expect(() => {
      parseRSSBriefing({
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
            importance: 'critical',
            source_item_count: 3,
            source_feed_count: 2,
          },
        ],
      });
    }).toThrow('Invalid rss briefing.highlights[0].importance: unexpected value "critical"');
  });

  it('ignores unknown fields on the rss briefing payload', () => {
    expect(parseRSSBriefing({
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
          extra_highlight_field: 'ignored',
        },
      ],
      extra: true,
    })).toEqual({
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
    });
  });
});
