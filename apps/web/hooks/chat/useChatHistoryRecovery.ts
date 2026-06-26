import { useCallback, useMemo, useRef } from 'react';
import { parseAgentErrorPayload } from '@/lib/api/agent/parser';
import { getSession } from '@/lib/api/sessions/api';
import { streamSessionEvents, toAgentStreamEvent } from '@/lib/api/sessions/events';
import {
  projectRecoveredTurnEventRunState,
  projectTurnDraftRunRecovery,
} from '@/lib/chat-stream/historyRecovery';
import { projectAgentEvent } from '@/lib/chatRuntime/eventProjector';
import {
  createChatRuntimeStateFromDraft,
  type ChatRuntimeState,
} from '@/lib/chatRuntime/runtimeState';
import { mapSessionMessagesToChat } from '@/lib/chatMessages';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SessionPushEvent, SessionTurnDraft } from '@/lib/types';
import { applyHistoryPageState, mergeLatestCommittedMessages } from './chatHistoryMerge';
import type { ActiveAgentRun, ChatStateControls } from './types';

const RECOVERY_HISTORY_LIMIT = 100;

interface UseChatHistoryRecoveryOptions {
  applyRuntimeActions: ChatStateControls['applyRuntimeActions'];
  beginHistorySync: ChatStateControls['beginHistorySync'];
  endHistorySync: ChatStateControls['endHistorySync'];
  hydrateTurnDraft: ChatStateControls['hydrateTurnDraft'];
  setActiveRun: ChatStateControls['setActiveRun'];
  setChatError: ChatStateControls['setChatError'];
  setCommittedMessages: ChatStateControls['setCommittedMessages'];
  setHasOlderHistory: ChatStateControls['setHasOlderHistory'];
  setLoading: ChatStateControls['setLoading'];
  setNextHistoryBefore: ChatStateControls['setNextHistoryBefore'];
  setStopPending: ChatStateControls['setStopPending'];
}

export function useChatHistoryRecovery(options: UseChatHistoryRecoveryOptions) {
  const { copy } = useWebLocale();
  const {
    applyRuntimeActions,
    beginHistorySync,
    endHistorySync,
    hydrateTurnDraft,
    setActiveRun,
    setChatError,
    setCommittedMessages,
    setHasOlderHistory,
    setLoading,
    setNextHistoryBefore,
    setStopPending,
  } = options;
  const historyPageStateTarget = useMemo(() => ({
    setHasOlderHistory,
    setNextHistoryBefore,
  }), [setHasOlderHistory, setNextHistoryBefore]);
  const recoveryRunRef = useRef<Record<string, ActiveAgentRun>>({});
  const stopRecoveredRun = useCallback((sessionId: string) => {
    const key = sessionId.trim();
    recoveryRunRef.current[key]?.abortController?.abort();
    delete recoveryRunRef.current[key];
  }, []);

  const syncRecoveredTurn = useCallback(async (sessionId: string, messageText?: string) => {
    stopRecoveredRun(sessionId);
    beginHistorySync(sessionId);
    try {
      const detail = await getSession(sessionId, { limit: RECOVERY_HISTORY_LIMIT });
      applyHistoryPageState(detail, sessionId, historyPageStateTarget);
      hydrateTurnDraft(sessionId, detail.turn_draft ?? null);
      setChatError(sessionId,
        messageText ?? (detail.turn_draft?.status === 'error' ? (detail.turn_draft.error ?? '') : ''),
      );
      setCommittedMessages(sessionId, (previous) => {
        return mergeLatestCommittedMessages(previous, mapSessionMessagesToChat(detail.id, detail.messages));
      });
    } catch (error) {
      setChatError(sessionId, toErrorMessage(error, copy.system.genericRequestFailed));
    } finally {
      setActiveRun(sessionId, null);
      setLoading(sessionId, false);
      setStopPending(sessionId, false);
      endHistorySync(sessionId);
    }
  }, [
    beginHistorySync,
    copy.system.genericRequestFailed,
    endHistorySync,
    historyPageStateTarget,
    hydrateTurnDraft,
    setActiveRun,
    setChatError,
    setCommittedMessages,
    setLoading,
    setStopPending,
    stopRecoveredRun,
  ]);

  const applyRecoveredEvent = useCallback((runtime: ChatRuntimeState, event: SessionPushEvent) => {
    const eventRunState = projectRecoveredTurnEventRunState(event.type);
    if (eventRunState.projectRuntimeEvent) {
      applyRuntimeActions(runtime.sessionId, projectAgentEvent({
        event: toAgentStreamEvent(event),
        runtime,
      }));
    }
    return eventRunState.historySync;
  }, [applyRuntimeActions]);

  const recoverTurnDraft = useCallback((sessionId: string, draft: SessionTurnDraft | null | undefined) => {
    stopRecoveredRun(sessionId);
    hydrateTurnDraft(sessionId, draft);
    const recovery = projectTurnDraftRunRecovery(sessionId, draft);
    setStopPending(sessionId, recovery.stopPending);
    setActiveRun(sessionId, null);
    setLoading(sessionId, recovery.loading);
    if (recovery.chatError !== undefined) {
      setChatError(sessionId, recovery.chatError);
    }

    if (recovery.phase !== 'streaming' || !draft || !recovery.activeRun) {
      return;
    }

    const runtime = createChatRuntimeStateFromDraft(draft, sessionId);
    const abortController = new AbortController();
    const recoveredRun = { ...recovery.activeRun, abortController };
    recoveryRunRef.current[sessionId.trim()] = recoveredRun;
    setActiveRun(sessionId, recoveredRun);

    void streamSessionEvents({
      onEvent: async (event) => {
        const historySync = applyRecoveredEvent(runtime, event);
        if (historySync === 'sync_error') {
          await syncRecoveredTurn(sessionId, parseAgentErrorPayload(event.payload).message);
          return;
        }
        if (historySync === 'sync') {
          await syncRecoveredTurn(sessionId);
        }
      },
      sessionId,
      signal: abortController.signal,
    }).catch((error) => {
      if (abortController.signal.aborted) {
        return;
      }
      setActiveRun(sessionId, null);
      setLoading(sessionId, false);
      setChatError(sessionId, toErrorMessage(error, copy.system.genericRequestFailed));
    });
  }, [
    applyRecoveredEvent,
    copy.system.genericRequestFailed,
    hydrateTurnDraft,
    setActiveRun,
    setChatError,
    setLoading,
    setStopPending,
    stopRecoveredRun,
    syncRecoveredTurn,
  ]);

  return {
    recoverTurnDraft,
    stopRecoveredRun,
  };
}
