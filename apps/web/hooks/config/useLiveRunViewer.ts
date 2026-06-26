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
  reconcileCardsWithRunStatus,
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
  const { cards, streamError } = useLiveRunCards(run);
  const { followLatest, selectedCard, selectCard } = useLiveRunSelection(cards, run.run_id);
  const { output, sourceSessionError } = useLiveRunOutput(selectedCard, run.run_id);

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

function useLiveRunCards(run: TaskRunLog): {
  cards: LiveTaskRunCard[];
  streamError: string;
} {
  const [cards, setCards] = useState<LiveTaskRunCard[]>(() => hydrateLiveTaskRunCards(run.run_cards));
  const [streamError, setStreamError] = useState('');

  useEffect(() => {
    setCards([]);
    setStreamError('');
  }, [run.run_id]);

  useEffect(() => {
    setCards((current) => reconcileCardsWithRunStatus(
      mergeLiveTaskRunCards(current, run.run_cards),
      run.status,
      {
        finished_at: run.finished_at,
        preview: run.response_preview,
        error: run.error,
      },
    ));
  }, [run.error, run.finished_at, run.response_preview, run.run_cards, run.status]);

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

  return { cards, streamError };
}

function useLiveRunSelection(cards: LiveTaskRunCard[], runID: string): {
  followLatest: boolean;
  selectedCard: LiveTaskRunCard | null;
  selectCard: (cardId: string) => void;
} {
  const [followLatest, setFollowLatest] = useState(true);
  const [selectedCardId, setSelectedCardId] = useState('');

  useEffect(() => {
    setFollowLatest(true);
    setSelectedCardId('');
  }, [runID]);

  const selectedCard = useMemo(() => {
    return resolveSelectedCard(cards, selectedCardId);
  }, [cards, selectedCardId]);

  useEffect(() => {
    syncSelectedCard({
      cards,
      followLatest,
      selectedCard,
      selectedCardId,
      setSelectedCardId,
    });
  }, [cards, followLatest, selectedCard, selectedCardId]);

  const selectCard = useCallback((cardId: string) => {
    setSelectedCardId(cardId);
    const latest = latestCreatedCard(cards);
    setFollowLatest(Boolean(latest && latest.card_id === cardId));
  }, [cards]);

  return { followLatest, selectedCard, selectCard };
}

function useLiveRunOutput(selectedCard: LiveTaskRunCard | null, runID: string): {
  output: TaskRunCardOutput;
  sourceSessionError: string;
} {
  const [sourceSessionError, setSourceSessionError] = useState('');
  const [sourceSessions, setSourceSessions] = useState<Record<string, SessionDetail>>({});
  const selectedSessionId = useMemo(() => resolveSelectedSessionId(selectedCard), [selectedCard]);
  const refreshToken = `${runID}:${selectedSessionId}:${selectedCard?.status ?? ''}:${selectedCard?.finished_at ?? ''}`;

  useEffect(() => {
    setSourceSessionError('');
    setSourceSessions({});
  }, [runID]);

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

  const output = useMemo(
    () => buildTaskRunCardOutput(selectedCard, selectedSessionId ? sourceSessions[selectedSessionId] ?? null : null),
    [selectedCard, selectedSessionId, sourceSessions],
  );

  return { output, sourceSessionError };
}

function resolveSelectedCard(cards: LiveTaskRunCard[], selectedCardId: string): LiveTaskRunCard | null {
  if (!cards.length) {
    return null;
  }

  const matched = selectedCardId
    ? cards.find((card) => card.card_id === selectedCardId)
    : null;

  return matched ?? latestActiveCard(cards) ?? latestCreatedCard(cards);
}

function syncSelectedCard(options: {
  cards: LiveTaskRunCard[];
  followLatest: boolean;
  selectedCard: LiveTaskRunCard | null;
  selectedCardId: string;
  setSelectedCardId: (value: string) => void;
}): void {
  const { cards, followLatest, selectedCard, selectedCardId, setSelectedCardId } = options;

  if (!cards.length) {
    clearSelectedCardId(selectedCardId, setSelectedCardId);
    return;
  }
  if (!followLatest || isStickyTerminalCard(selectedCard)) {
    return;
  }

  const next = latestActiveCard(cards) ?? latestCreatedCard(cards);
  if (next && next.card_id !== selectedCardId) {
    setSelectedCardId(next.card_id);
  }
}

function clearSelectedCardId(
  selectedCardId: string,
  setSelectedCardId: (value: string) => void,
): void {
  if (selectedCardId) {
    setSelectedCardId('');
  }
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
        final_text: payload.final_text,
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
  if (card.source_events.length > 0) {
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
