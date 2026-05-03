import { useCallback, useRef } from 'react';
import { parseAgentErrorPayload } from '@/lib/api/agent/parser';
import { streamSessionEvents, toAgentStreamEvent } from '@/lib/api/sessions/events';
import { getSession } from '@/lib/api/sessions/api';
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
  clearPendingQuestions: ChatStateControls['clearPendingQuestions'];
  clearStreamingState: ChatStateControls['clearStreamingState'];
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
  const recoveryRunRef = useRef<ActiveAgentRun | null>(null);
  const stopRecoveredRun = useCallback(() => {
    recoveryRunRef.current?.abortController?.abort();
    recoveryRunRef.current = null;
  }, []);

  const syncRecoveredTurn = useCallback(async (
    sessionId: string,
    messageText?: string,
    preservePendingQuestion = false,
  ) => {
    stopRecoveredRun();
    options.beginHistorySync();
    try {
      if (messageText) {
        options.setChatError(messageText);
      }
      const detail = await getSession(sessionId, { limit: RECOVERY_HISTORY_LIMIT });
      applyHistoryPage(detail, options.setHasOlderHistory, options.setNextHistoryBefore);
      if (!preservePendingQuestion) {
        options.clearPendingQuestions();
      }
      options.clearStreamingState();
      options.hydrateTurnDraft(null);
      options.setCommittedMessages((previous) => {
        return mergeLatestCommittedMessages(previous, mapSessionMessagesToChat(detail.id, detail.messages));
      });
    } catch (error) {
      options.setChatError(toErrorMessage(error, copy.system.genericRequestFailed));
    } finally {
      options.setActiveRun(null);
      options.setLoading(false);
      options.setStopPending(false);
      options.endHistorySync();
    }
  }, [copy.system.genericRequestFailed, options, stopRecoveredRun]);

  const applyRecoveredEvent = useCallback((runtime: ChatRuntimeState, event: SessionPushEvent) => {
    options.applyRuntimeActions(projectAgentEvent({
      event: toAgentStreamEvent(event),
      runtime,
    }));
  }, [options]);

  const recoverTurnDraft = useCallback((sessionId: string, draft: SessionTurnDraft | null | undefined) => {
    stopRecoveredRun();
    options.hydrateTurnDraft(draft);
    if (!draft) {
      options.setActiveRun(null);
      options.setLoading(false);
      return;
    }

    const abortController = new AbortController();
    const runtime = createChatRuntimeStateFromDraft(draft, sessionId);
    const recoveredRun = { abortController, sessionId, traceId: draft.trace_id };
    recoveryRunRef.current = recoveredRun;
    options.setActiveRun(recoveredRun);
    options.setLoading(true);
    options.setStopPending(false);

    void streamSessionEvents({
      onEvent: async (event) => {
        switch (event.type) {
          case 'assistant_message':
            await syncRecoveredTurn(sessionId);
            return;
          case 'awaiting_human':
            applyRecoveredEvent(runtime, event);
            await syncRecoveredTurn(sessionId, undefined, true);
            return;
          case 'completion_delta':
          case 'tool_call_started':
          case 'tool_call_finished':
          case 'run_started':
          case 'done':
            applyRecoveredEvent(runtime, event);
            return;
          case 'error':
            applyRecoveredEvent(runtime, event);
            await syncRecoveredTurn(sessionId, parseAgentErrorPayload(event.payload).message);
            return;
          default:
            return;
        }
      },
      sessionId,
      signal: abortController.signal,
    }).catch((error) => {
      if (abortController.signal.aborted) {
        return;
      }
      options.setActiveRun(null);
      options.setLoading(false);
      options.setChatError(toErrorMessage(error, copy.system.genericRequestFailed));
    });
  }, [applyRecoveredEvent, copy.system.genericRequestFailed, options, stopRecoveredRun, syncRecoveredTurn]);

  return {
    recoverTurnDraft,
    stopRecoveredRun,
  };
}

function applyHistoryPage(
  detail: SessionDetail,
  setHasOlderHistory: ChatStateControls['setHasOlderHistory'],
  setNextHistoryBefore: ChatStateControls['setNextHistoryBefore'],
): void {
  setHasOlderHistory(detail.page.has_more_before);
  setNextHistoryBefore(detail.page.next_before ?? null);
}
