import type { RSSBriefing } from '@/lib/types';
import { requestJSON, requestOptionalJSON } from '@/lib/api/client';
import { parseRSSBriefing } from '@/lib/api/rss/parser';

export interface BuildRSSBriefingInput {
  window_hours?: number;
  group_limit?: number;
  item_limit?: number;
  items_per_group?: number;
  highlights_limit?: number;
}

export async function getRSSBriefing(): Promise<RSSBriefing | null> {
  return requestOptionalJSON('/api/rss/briefing', {}, parseRSSBriefing);
}

export async function buildRSSBriefing(
  input: BuildRSSBriefingInput = {},
): Promise<RSSBriefing> {
  return requestJSON('/api/rss/briefing', {
    method: 'POST',
    body: JSON.stringify(input),
  }, parseRSSBriefing);
}
