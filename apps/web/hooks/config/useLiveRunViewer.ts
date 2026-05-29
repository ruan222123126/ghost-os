import { useEffect, useState } from 'react';
import { getSession } from '@/lib/api/sessions/api';
import { streamSessionEvents } from '@/lib/api/sessions/events';
import type { SessionDetail, SessionPushEvent } from '@/lib/types';

const MAX_LIVE_EVENTS = 500;
const SESSION_POLL_INTERVAL_MS = 800;
const SESSION_MESSAGE_LIMIT = 50;

interface UseLiveRunViewerOptions {
  sessionId: string;
}

interface UseLiveRunViewerResult {
  events: SessionPushEvent[];
  session: SessionDetail | null;
  streamError: string;
  snapshotError: string;
}

export function useLiveRunViewer(options: UseLiveRunViewerOptions): UseLiveRunViewerResult {
  const { sessionId } = options;
  const [events, setEvents] = useState<SessionPushEvent[]>([]);
  const [session, setSession] = useState<SessionDetail | null>(null);
  const [streamError, setStreamError] = useState('');
  const [snapshotError, setSnapshotError] = useState('');

  useEffect(() => subscribeSessionEvents(sessionId, setEvents, setStreamError), [sessionId]);
  useEffect(() => pollSessionSnapshot(sessionId, setSession, setSnapshotError), [sessionId]);

  return {
    events,
    session,
    streamError,
    snapshotError,
  };
}

function subscribeSessionEvents(
  sessionId: string,
  setEvents: (updater: (events: SessionPushEvent[]) => SessionPushEvent[]) => void,
  setError: (message: string) => void,
) {
  const controller = new AbortController();
  void streamSessionEvents({
    sessionId,
    signal: controller.signal,
    onEvent: (event) => {
      setError('');
      setEvents((events) => appendLiveEvent(events, event));
    },
  }).catch((error: unknown) => {
    if (!controller.signal.aborted) {
      setError(errorMessage(error));
    }
  });
  return () => controller.abort();
}

function pollSessionSnapshot(
  sessionId: string,
  setSession: (session: SessionDetail) => void,
  setError: (message: string) => void,
) {
  let active = true;
  let inFlight = false;
  const load = async () => {
    if (inFlight) {
      return;
    }
    inFlight = true;
    try {
      const detail = await getSession(sessionId, { limit: SESSION_MESSAGE_LIMIT });
      if (active) {
        setSession(detail);
        setError('');
      }
    } catch (error) {
      if (active) {
        setError(errorMessage(error));
      }
    } finally {
      inFlight = false;
    }
  };
  void load();
  const timer = setInterval(() => void load(), SESSION_POLL_INTERVAL_MS);
  return () => {
    active = false;
    clearInterval(timer);
  };
}

function appendLiveEvent(events: SessionPushEvent[], event: SessionPushEvent): SessionPushEvent[] {
  const next = [...events, event];
  if (next.length <= MAX_LIVE_EVENTS) {
    return next;
  }
  return next.slice(next.length - MAX_LIVE_EVENTS);
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}
