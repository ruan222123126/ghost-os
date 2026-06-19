import { useCallback, useRef } from 'react';
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
import type { SessionDetail, SessionPushEvent, SessionTurnDraft } from '@/lib/types';
import { mergeLatestCommittedMessages } from './chatHistoryMerge';
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
  const recoveryRunRef = useRef<Record<string, ActiveAgentRun>>({});
  const stopRecoveredRun = useCallback((sessionId: string) => {
    const key = sessionId.trim();
    recoveryRunRef.current[key]?.abortController?.abort();
    delete recoveryRunRef.current[key];
  }, []);

  const syncRecoveredTurn = useCallback(async (sessionId: string, messageText?: string) => {
    stopRecoveredRun(sessionId);
    options.beginHistorySync(sessionId);
    try {
      const detail = await getSession(sessionId, { limit: RECOVERY_HISTORY_LIMIT });
      applyHistoryPage(detail, sessionId, options.setHasOlderHistory, options.setNextHistoryBefore);
      options.hydrateTurnDraft(sessionId, detail.turn_draft ?? null);
      options.setChatError(sessionId,
        messageText ?? (detail.turn_draft?.status === 'error' ? (detail.turn_draft.error ?? '') : ''),
      );
      options.setCommittedMessages(sessionId, (previous) => {
        return mergeLatestCommittedMessages(previous, mapSessionMessagesToChat(detail.id, detail.messages));
      });
    } catch (error) {
      options.setChatError(sessionId, toErrorMessage(error, copy.system.genericRequestFailed));
    } finally {
      options.setActiveRun(sessionId, null);
      options.setLoading(sessionId, false);
      options.setStopPending(sessionId, false);
      options.endHistorySync(sessionId);
    }
  }, [copy.system.genericRequestFailed, options, stopRecoveredRun]);

  const applyRecoveredEvent = useCallback((runtime: ChatRuntimeState, event: SessionPushEvent) => {
    const eventRunState = projectRecoveredTurnEventRunState(event.type);
    if (eventRunState.projectRuntimeEvent) {
      options.applyRuntimeActions(runtime.sessionId, projectAgentEvent({
        event: toAgentStreamEvent(event),
        runtime,
      }));
    }
    return eventRunState.historySync;
  }, [options]);

  const recoverTurnDraft = useCallback((sessionId: string, draft: SessionTurnDraft | null | undefined) => {
    stopRecoveredRun(sessionId);
    options.hydrateTurnDraft(sessionId, draft);
    const recovery = projectTurnDraftRunRecovery(sessionId, draft);
    options.setStopPending(sessionId, recovery.stopPending);
    options.setActiveRun(sessionId, null);
    options.setLoading(sessionId, recovery.loading);
    if (recovery.chatError !== undefined) {
      options.setChatError(sessionId, recovery.chatError);
    }

    if (recovery.phase !== 'streaming' || !draft || !recovery.activeRun) {
      return;
    }

    const runtime = createChatRuntimeStateFromDraft(draft, sessionId);
    const abortController = new AbortController();
    const recoveredRun = { ...recovery.activeRun, abortController };
    recoveryRunRef.current[sessionId.trim()] = recoveredRun;
    options.setActiveRun(sessionId, recoveredRun);

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
      options.setActiveRun(sessionId, null);
      options.setLoading(sessionId, false);
      options.setChatError(sessionId, toErrorMessage(error, copy.system.genericRequestFailed));
    });
  }, [applyRecoveredEvent, copy.system.genericRequestFailed, options, stopRecoveredRun, syncRecoveredTurn]);

  return {
    recoverTurnDraft,
    stopRecoveredRun,
  };
}

function applyHistoryPage(
  detail: SessionDetail,
  sessionId: string,
  setHasOlderHistory: ChatStateControls['setHasOlderHistory'],
  setNextHistoryBefore: ChatStateControls['setNextHistoryBefore'],
): void {
  setHasOlderHistory(sessionId, detail.page.has_more_before);
  setNextHistoryBefore(sessionId, detail.page.next_before ?? null);
}
