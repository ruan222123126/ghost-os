import type { AgentStreamEvent } from '@/lib/types';
import type { TaskRunCard } from '@/lib/taskRunCards';

export interface LiveTaskRunCard extends TaskRunCard {
  live_source_session_id?: string;
  source_events: AgentStreamEvent[];
}

export function hydrateLiveTaskRunCards(
  cards: TaskRunCard[] | undefined,
): LiveTaskRunCard[] {
  if (!cards?.length) {
    return [];
  }
  return cards
    .map((card) => ({
      ...card,
      live_source_session_id: card.source_session_id || resolveSourceSessionIDFromEvents(card.source_events),
      source_events: mergeCardSourceEvents(card.source_events),
    }))
    .sort(compareCardsByStartTime);
}

export function mergeLiveTaskRunCards(
  current: LiveTaskRunCard[],
  cards: TaskRunCard[] | undefined,
): LiveTaskRunCard[] {
  if (!cards?.length) {
    return [];
  }
  const currentByID = new Map(current.map((card) => [card.card_id, card]));
  return cards
    .map((card) => {
      const existing = currentByID.get(card.card_id);
      const sourceEvents = mergeCardSourceEvents(card.source_events, existing?.source_events);
      return {
        ...card,
        live_source_session_id: card.source_session_id
          || resolveSourceSessionIDFromEvents(sourceEvents)
          || existing?.live_source_session_id,
        source_events: sourceEvents,
      };
    })
    .sort(compareCardsByStartTime);
}

export function applyStartedCard(
  cards: LiveTaskRunCard[],
  card: TaskRunCard,
): LiveTaskRunCard[] {
  const next = cards.filter((current) => current.card_id !== card.card_id);
  next.push({
    ...card,
    live_source_session_id: card.source_session_id || resolveSourceSessionIDFromEvents(card.source_events),
    source_events: mergeCardSourceEvents(card.source_events),
  });
  next.sort(compareCardsByStartTime);
  return next;
}

export function applyCardEvent(
  cards: LiveTaskRunCard[],
  cardID: string,
  event: AgentStreamEvent,
  sourceSessionId?: string,
): LiveTaskRunCard[] {
  return cards.map((card) => {
    if (card.card_id !== cardID) {
      return card;
    }
    return {
      ...card,
      live_source_session_id: sourceSessionId?.trim() || card.live_source_session_id,
      source_events: mergeCardSourceEvents(card.source_events, [event]),
    };
  });
}

export function applyFinishedCard(
  cards: LiveTaskRunCard[],
  patch: Pick<LiveTaskRunCard, 'card_id' | 'status' | 'finished_at' | 'preview' | 'error' | 'live_source_session_id'>,
): LiveTaskRunCard[] {
  return cards.map((card) => {
    if (card.card_id !== patch.card_id) {
      return card;
    }
    return {
      ...card,
      status: patch.status,
      finished_at: patch.finished_at,
      preview: patch.preview,
      error: patch.error,
      live_source_session_id: patch.live_source_session_id || card.live_source_session_id,
    };
  });
}

export function latestCreatedCard(cards: LiveTaskRunCard[]): LiveTaskRunCard | null {
  return cards.at(-1) ?? null;
}

export function latestActiveCard(cards: LiveTaskRunCard[]): LiveTaskRunCard | null {
  for (let index = cards.length - 1; index >= 0; index -= 1) {
    const card = cards[index];
    if (isActiveCard(card)) {
      return card;
    }
  }
  return null;
}

export function isStickyTerminalCard(card: LiveTaskRunCard | null | undefined): boolean {
  const status = card?.status?.trim();
  return status === 'awaiting_human' || status === 'error' || status === 'cancelled';
}

export function isSummaryOnlyCard(card: LiveTaskRunCard): boolean {
  return card.kind === 'relay_round' && card.status !== 'running' && Boolean(card.final_text?.trim());
}

export function resolveCardSourceSessionId(card: LiveTaskRunCard | null | undefined): string {
  if (!card) {
    return '';
  }
  return card.live_source_session_id?.trim()
    || card.source_session_id?.trim()
    || resolveSourceSessionIDFromEvents(card.source_events)
    || '';
}

function isActiveCard(card: LiveTaskRunCard): boolean {
  const status = card.status?.trim() || 'running';
  return status === 'running';
}

function mergeCardSourceEvents(...lists: (AgentStreamEvent[] | undefined)[]): AgentStreamEvent[] {
  const merged: AgentStreamEvent[] = [];
  const seen = new Set<string>();
  for (const list of lists) {
    for (const event of list ?? []) {
      const eventId = event.id.trim();
      const dedupeKey = eventId || eventKey(event);
      if (seen.has(dedupeKey)) {
        continue;
      }
      seen.add(dedupeKey);
      merged.push(event);
    }
  }
  return merged;
}

function resolveSourceSessionIDFromEvents(events: AgentStreamEvent[] | undefined): string {
  for (let index = (events?.length ?? 0) - 1; index >= 0; index -= 1) {
    const sessionId = events?.[index]?.session_id?.trim();
    if (sessionId) {
      return sessionId;
    }
  }
  return '';
}

function eventKey(event: AgentStreamEvent): string {
  return [
    event.trace_id,
    event.step_id,
    event.turn,
    event.type,
    event.at ?? '',
  ].join(':');
}

function compareCardsByStartTime(left: LiveTaskRunCard, right: LiveTaskRunCard): number {
  const leftTime = Date.parse(left.started_at ?? '') || 0;
  const rightTime = Date.parse(right.started_at ?? '') || 0;
  if (leftTime !== rightTime) {
    return leftTime - rightTime;
  }
  return left.card_id.localeCompare(right.card_id);
}
