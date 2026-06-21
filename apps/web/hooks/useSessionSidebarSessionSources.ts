'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { getSessionSources } from '@/lib/api/sessions/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import {
  sessionSourceAssignmentsFromPayload,
  type SessionSourceAssignments,
} from '@/lib/sessionSidebarSessionSources';
import type { SessionMetadata } from '@/lib/types';

interface UseSessionSidebarSessionSourcesOptions {
  sessions: SessionMetadata[];
  requestFailedText: string;
}

interface UseSessionSidebarSessionSourcesResult {
  assignments: SessionSourceAssignments;
  hiddenSessionIDs: string[];
  error: string;
}

export function useSessionSidebarSessionSources(
  options: UseSessionSidebarSessionSourcesOptions,
): UseSessionSidebarSessionSourcesResult {
  const {
    sessions,
    requestFailedText,
  } = options;
  const [assignments, setAssignments] = useState<SessionSourceAssignments>({});
  const [hiddenSessionIDs, setHiddenSessionIDs] = useState<string[]>([]);
  const [error, setError] = useState('');
  const sessionIDKey = useMemo(() => {
    return sessions.map((session) => session.id).join('\u0000');
  }, [sessions]);
  const knownSessionIDs = useMemo(() => {
    return sessionIDKey.length === 0 ? new Set<string>() : new Set(sessionIDKey.split('\u0000'));
  }, [sessionIDKey]);

  const reload = useCallback(async () => {
    try {
      const resolution = await getSessionSources();
      setAssignments(filterKnownAssignments(
        sessionSourceAssignmentsFromPayload(resolution.assignments),
        knownSessionIDs,
      ));
      setHiddenSessionIDs(filterKnownSessionIDs(resolution.hidden_session_ids, knownSessionIDs));
      setError('');
    } catch (error) {
      setAssignments({});
      setHiddenSessionIDs([]);
      setError(toErrorMessage(error, requestFailedText));
    }
  }, [knownSessionIDs, requestFailedText]);

  useEffect(() => {
    ignorePromise(reload());
  }, [reload]);

  return { assignments, hiddenSessionIDs, error };
}

function filterKnownAssignments(
  assignments: SessionSourceAssignments,
  knownSessionIDs: ReadonlySet<string>,
): SessionSourceAssignments {
  const filtered: SessionSourceAssignments = {};
  for (const [sessionID, source] of Object.entries(assignments)) {
    if (knownSessionIDs.has(sessionID)) {
      filtered[sessionID] = source;
    }
  }
  return filtered;
}

function filterKnownSessionIDs(
  candidateIDs: string[],
  knownSessionIDs: ReadonlySet<string>,
): string[] {
  return candidateIDs.filter((sessionID) => knownSessionIDs.has(sessionID));
}
