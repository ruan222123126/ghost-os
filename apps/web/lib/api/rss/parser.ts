import type { RSSBriefing, RSSBriefingHighlight } from '@/lib/types';
import {
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  pickKnownKeys,
  parseOptionalString,
  parseOptionalStringArray,
} from '@/lib/api/shared';

const RSS_BRIEFING_KEYS = [
  'id',
  'title',
  'summary',
  'generated_at',
  'saved_at',
  'window_hours',
  'scanned_groups',
  'highlight_count',
  'trace_id',
  'task_id',
  'highlights',
] as const;
const RSS_HIGHLIGHT_KEYS = [
  'rank',
  'group_id',
  'topic_label',
  'headline',
  'summary',
  'why_it_matters',
  'importance',
  'source_item_count',
  'source_feed_count',
  'tags',
] as const;
const RSS_IMPORTANCE_LEVELS = ['low', 'normal', 'high'] as const;

function parseRSSBriefingHighlight(value: unknown, label: string): RSSBriefingHighlight {
  const record = pickKnownKeys(expectRecord(value, label), RSS_HIGHLIGHT_KEYS);

  return {
    rank: expectNumber(record.rank, `${label}.rank`),
    group_id: expectString(record.group_id, `${label}.group_id`),
    topic_label: parseOptionalString(record.topic_label, `${label}.topic_label`),
    headline: expectString(record.headline, `${label}.headline`),
    summary: parseOptionalString(record.summary, `${label}.summary`),
    why_it_matters: parseOptionalString(record.why_it_matters, `${label}.why_it_matters`),
    importance: expectStringEnum(record.importance, RSS_IMPORTANCE_LEVELS, `${label}.importance`),
    source_item_count: expectNumber(record.source_item_count, `${label}.source_item_count`),
    source_feed_count: expectNumber(record.source_feed_count, `${label}.source_feed_count`),
    tags: parseOptionalStringArray(record.tags, `${label}.tags`),
  };
}

export function parseRSSBriefing(payload: unknown): RSSBriefing {
  const record = pickKnownKeys(expectRecord(payload, 'rss briefing'), RSS_BRIEFING_KEYS);
  if (!Array.isArray(record.highlights)) {
    throw new Error('Invalid rss briefing.highlights: expected array');
  }

  return {
    id: parseOptionalString(record.id, 'rss briefing.id'),
    title: expectString(record.title, 'rss briefing.title'),
    summary: parseOptionalString(record.summary, 'rss briefing.summary'),
    generated_at: expectString(record.generated_at, 'rss briefing.generated_at'),
    saved_at: parseOptionalString(record.saved_at, 'rss briefing.saved_at'),
    window_hours: expectNumber(record.window_hours, 'rss briefing.window_hours'),
    scanned_groups: expectNumber(record.scanned_groups, 'rss briefing.scanned_groups'),
    highlight_count: expectNumber(record.highlight_count, 'rss briefing.highlight_count'),
    trace_id: parseOptionalString(record.trace_id, 'rss briefing.trace_id'),
    task_id: parseOptionalString(record.task_id, 'rss briefing.task_id'),
    highlights: record.highlights.map((item, index) => {
      return parseRSSBriefingHighlight(item, `rss briefing.highlights[${index}]`);
    }),
  };
}
