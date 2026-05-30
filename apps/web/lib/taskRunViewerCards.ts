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
      live_source_session_id: card.source_session_id,
      source_events: [],
    }))
    .sort(compareCardsByStartTime);
}

export function applyStartedCard(
  cards: LiveTaskRunCard[],
  card: TaskRunCard,
): LiveTaskRunCard[] {
  const next = cards.filter((current) => current.card_id !== card.card_id);
  next.push({
    ...card,
    live_source_session_id: card.source_session_id,
    source_events: [],
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
      source_events: [...card.source_events, event],
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

function isActiveCard(card: LiveTaskRunCard): boolean {
  const status = card.status?.trim() || 'running';
  return status === 'running';
}

function compareCardsByStartTime(left: LiveTaskRunCard, right: LiveTaskRunCard): number {
  const leftTime = Date.parse(left.started_at ?? '') || 0;
  const rightTime = Date.parse(right.started_at ?? '') || 0;
  if (leftTime !== rightTime) {
    return leftTime - rightTime;
  }
  return left.card_id.localeCompare(right.card_id);
}
