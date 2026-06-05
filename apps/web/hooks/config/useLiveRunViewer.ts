import { useCallback, useEffect, useMemo, useState } from 'react';
import { getFullSession } from '@/lib/api/sessions/api';
import { streamSessionEvents } from '@/lib/api/sessions/events';
import {
  parseTaskRunCardEventPayload,
  parseTaskRunCardFinishedPayload,
  parseTaskRunCardStartedPayload,
} from '@/lib/api/sessions/taskRunCards';
import { buildTaskRunCardOutput, type TaskRunCardOutput } from '@/lib/taskRunViewerOutput';
import {
  applyCardEvent,
  applyFinishedCard,
  applyStartedCard,
  hydrateLiveTaskRunCards,
  isStickyTerminalCard,
  latestActiveCard,
  latestCreatedCard,
  mergeLiveTaskRunCards,
  resolveCardSourceSessionId,
  type LiveTaskRunCard,
} from '@/lib/taskRunViewerCards';
import type { SessionDetail, TaskRunLog } from '@/lib/types';

const SESSION_PAGE_LIMIT = 200;

interface UseLiveRunViewerOptions {
  run: TaskRunLog;
}

interface UseLiveRunViewerResult {
  cards: LiveTaskRunCard[];
  followLatest: boolean;
  output: TaskRunCardOutput;
  selectedCard: LiveTaskRunCard | null;
  selectCard: (cardId: string) => void;
  sourceSessionError: string;
  streamError: string;
}

export function useLiveRunViewer(options: UseLiveRunViewerOptions): UseLiveRunViewerResult {
  const { run } = options;
  const [cards, setCards] = useState<LiveTaskRunCard[]>(() => hydrateLiveTaskRunCards(run.run_cards));
  const [followLatest, setFollowLatest] = useState(true);
  const [selectedCardId, setSelectedCardId] = useState('');
  const [streamError, setStreamError] = useState('');
  const [sourceSessionError, setSourceSessionError] = useState('');
  const [sourceSessions, setSourceSessions] = useState<Record<string, SessionDetail>>({});

  useEffect(() => {
    setCards([]);
    setFollowLatest(true);
    setSelectedCardId('');
    setStreamError('');
    setSourceSessionError('');
    setSourceSessions({});
  }, [run.run_id]);

  useEffect(() => {
    setCards((current) => mergeLiveTaskRunCards(current, run.run_cards));
  }, [run.run_cards]);

  useEffect(() => {
    const sessionId = run.session_id_output?.trim();
    if (!sessionId) {
      return undefined;
    }
    const controller = new AbortController();
    void streamSessionEvents({
      sessionId,
      signal: controller.signal,
      onEvent: (event) => {
        setStreamError('');
        setCards((current) => applyViewerEvent(current, event));
      },
    }).catch((error: unknown) => {
      if (!controller.signal.aborted) {
        setStreamError(errorMessage(error));
      }
    });
    return () => controller.abort();
  }, [run.session_id_output]);

  const selectedCard = useMemo(() => {
    if (!cards.length) {
      return null;
    }
    if (selectedCardId) {
      const matched = cards.find((card) => card.card_id === selectedCardId);
      if (matched) {
        return matched;
      }
    }
    return latestActiveCard(cards) ?? latestCreatedCard(cards);
  }, [cards, selectedCardId]);

  useEffect(() => {
    if (!cards.length) {
      if (selectedCardId) {
        setSelectedCardId('');
      }
      return;
    }
    if (!followLatest || isStickyTerminalCard(selectedCard)) {
      return;
    }
    const next = latestActiveCard(cards) ?? latestCreatedCard(cards);
    if (next && next.card_id !== selectedCardId) {
      setSelectedCardId(next.card_id);
    }
  }, [cards, followLatest, selectedCard, selectedCardId]);

  const selectedSessionId = useMemo(() => resolveSelectedSessionId(selectedCard), [selectedCard]);
  const refreshToken = `${selectedSessionId}:${selectedCard?.status ?? ''}:${selectedCard?.finished_at ?? ''}`;

  useEffect(() => {
    if (!selectedSessionId) {
      setSourceSessionError('');
      return;
    }
    let active = true;
    void getFullSession(selectedSessionId, SESSION_PAGE_LIMIT)
      .then((detail) => {
        if (!active) {
          return;
        }
        setSourceSessions((current) => ({
          ...current,
          [selectedSessionId]: detail,
        }));
        setSourceSessionError('');
      })
      .catch((error: unknown) => {
        if (active) {
          setSourceSessionError(errorMessage(error));
        }
      });
    return () => {
      active = false;
    };
  }, [refreshToken, selectedSessionId]);

  const selectCard = useCallback((cardId: string) => {
    setSelectedCardId(cardId);
    const latest = latestCreatedCard(cards);
    setFollowLatest(Boolean(latest && latest.card_id === cardId));
  }, [cards]);

  const output = useMemo(
    () => buildTaskRunCardOutput(selectedCard, selectedSessionId ? sourceSessions[selectedSessionId] ?? null : null),
    [selectedCard, selectedSessionId, sourceSessions],
  );

  return {
    cards,
    followLatest,
    output,
    selectedCard,
    selectCard,
    sourceSessionError,
    streamError,
  };
}

function applyViewerEvent(cards: LiveTaskRunCard[], event: { type: string; payload: unknown }): LiveTaskRunCard[] {
  switch (event.type) {
    case 'task_run_card_started': {
      const payload = parseTaskRunCardStartedPayload(event.payload);
      return applyStartedCard(cards, {
        card_id: payload.card_id,
        run_id: payload.run_id,
        kind: payload.kind,
        title: payload.title,
        node_id: payload.node_id,
        node_type: payload.node_type,
        round: payload.round,
        iteration: payload.iteration,
        branch_id: payload.branch_id,
        source_session_id: payload.source_session_id,
        started_at: payload.started_at,
        status: 'running',
      });
    }
    case 'task_run_card_event': {
      const payload = parseTaskRunCardEventPayload(event.payload);
      return applyCardEvent(cards, payload.card_id, payload.source_event, payload.source_session_id);
    }
    case 'task_run_card_finished': {
      const payload = parseTaskRunCardFinishedPayload(event.payload);
      return applyFinishedCard(cards, {
        card_id: payload.card_id,
        status: payload.status,
        finished_at: payload.finished_at,
        preview: payload.preview,
        error: payload.error,
        live_source_session_id: payload.source_session_id,
      });
    }
    default:
      return cards;
  }
}

function resolveSelectedSessionId(card: LiveTaskRunCard | null): string {
  if (!card) {
    return '';
  }
  if (card.kind === 'relay_round' && card.status !== 'running' && card.source_events.length === 0) {
    return '';
  }
  return resolveCardSourceSessionId(card);
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}
