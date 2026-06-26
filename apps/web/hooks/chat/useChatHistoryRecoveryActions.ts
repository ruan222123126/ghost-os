import { useCallback, useMemo, useRef, type MutableRefObject } from 'react';
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
import type { SessionPushEvent, SessionTurnDraft } from '@/lib/types';
import { applyHistoryPageState, mergeLatestCommittedMessages } from './chatHistoryMerge';
import type { ActiveAgentRun, ChatStateControls } from './types';

const RECOVERY_HISTORY_LIMIT = 100;

export interface UseChatHistoryRecoveryActionsOptions {
  applyRuntimeActions: ChatStateControls['applyRuntimeActions'];
  beginHistorySync: ChatStateControls['beginHistorySync'];
  endHistorySync: ChatStateControls['endHistorySync'];
  hydrateTurnDraft: ChatStateControls['hydrateTurnDraft'];
  requestFailedText: string;
  setActiveRun: ChatStateControls['setActiveRun'];
  setChatError: ChatStateControls['setChatError'];
  setCommittedMessages: ChatStateControls['setCommittedMessages'];
  setHasOlderHistory: ChatStateControls['setHasOlderHistory'];
  setLoading: ChatStateControls['setLoading'];
  setNextHistoryBefore: ChatStateControls['setNextHistoryBefore'];
  setStopPending: ChatStateControls['setStopPending'];
}

type HistoryPageStateTarget = Parameters<typeof applyHistoryPageState>[2];

interface RecoveredTurnSyncOptions extends UseChatHistoryRecoveryActionsOptions {
  historyPageStateTarget: HistoryPageStateTarget;
  stopRecoveredRun: (sessionId: string) => void;
}

interface RecoverTurnDraftOptions extends UseChatHistoryRecoveryActionsOptions {
  applyRecoveredEvent: (runtime: ChatRuntimeState, event: SessionPushEvent) => ReturnType<typeof projectRecoveredTurnEventRunState>['historySync'];
  recoveryRunRef: MutableRefObject<Record<string, ActiveAgentRun>>;
  stopRecoveredRun: (sessionId: string) => void;
  syncRecoveredTurn: (sessionId: string, messageText?: string) => Promise<void>;
}

interface RecoveredStreamOptions extends Pick<RecoverTurnDraftOptions,
  | 'applyRecoveredEvent'
  | 'requestFailedText'
  | 'setActiveRun'
  | 'setChatError'
  | 'setLoading'
  | 'syncRecoveredTurn'
> {
  abortController: AbortController;
  runtime: ChatRuntimeState;
  sessionId: string;
}

export function useChatHistoryRecoveryActions(options: UseChatHistoryRecoveryActionsOptions) {
  const historyPageStateTarget = useMemo(() => ({
    setHasOlderHistory: options.setHasOlderHistory,
    setNextHistoryBefore: options.setNextHistoryBefore,
  }), [options.setHasOlderHistory, options.setNextHistoryBefore]);
  const recoveryRunRef = useRef<Record<string, ActiveAgentRun>>({});
  const stopRecoveredRun = useStopRecoveredRun(recoveryRunRef);
  const syncRecoveredTurn = useRecoveredTurnSync({
    ...options,
    historyPageStateTarget,
    stopRecoveredRun,
  });
  const applyRecoveredEvent = useRecoveredEventProjector(options.applyRuntimeActions);
  const recoverTurnDraft = useRecoverTurnDraft({
    ...options,
    applyRecoveredEvent,
    recoveryRunRef,
    stopRecoveredRun,
    syncRecoveredTurn,
  });

  return {
    recoverTurnDraft,
    stopRecoveredRun,
  };
}

function useStopRecoveredRun(recoveryRunRef: MutableRefObject<Record<string, ActiveAgentRun>>) {
  return useCallback((sessionId: string) => {
    const key = sessionId.trim();
    recoveryRunRef.current[key]?.abortController?.abort();
    delete recoveryRunRef.current[key];
  }, [recoveryRunRef]);
}

function useRecoveredTurnSync(options: RecoveredTurnSyncOptions) {
  const {
    beginHistorySync,
    endHistorySync,
    historyPageStateTarget,
    hydrateTurnDraft,
    requestFailedText,
    setActiveRun,
    setChatError,
    setCommittedMessages,
    setLoading,
    setStopPending,
    stopRecoveredRun,
  } = options;

  return useCallback(async (sessionId: string, messageText?: string) => {
    stopRecoveredRun(sessionId);
    beginHistorySync(sessionId);
    try {
      const detail = await getSession(sessionId, { limit: RECOVERY_HISTORY_LIMIT });
      applyHistoryPageState(detail, sessionId, historyPageStateTarget);
      hydrateTurnDraft(sessionId, detail.turn_draft ?? null);
      setChatError(sessionId, resolveRecoveredTurnError(detail.turn_draft, messageText));
      setCommittedMessages(sessionId, (previous) => {
        return mergeLatestCommittedMessages(previous, mapSessionMessagesToChat(detail.id, detail.messages));
      });
    } catch (error) {
      setChatError(sessionId, toErrorMessage(error, requestFailedText));
    } finally {
      setActiveRun(sessionId, null);
      setLoading(sessionId, false);
      setStopPending(sessionId, false);
      endHistorySync(sessionId);
    }
  }, [
    beginHistorySync,
    endHistorySync,
    historyPageStateTarget,
    hydrateTurnDraft,
    requestFailedText,
    setActiveRun,
    setChatError,
    setCommittedMessages,
    setLoading,
    setStopPending,
    stopRecoveredRun,
  ]);
}

function useRecoveredEventProjector(applyRuntimeActions: ChatStateControls['applyRuntimeActions']) {
  return useCallback((runtime: ChatRuntimeState, event: SessionPushEvent) => {
    const eventRunState = projectRecoveredTurnEventRunState(event.type);
    if (eventRunState.projectRuntimeEvent) {
      applyRuntimeActions(runtime.sessionId, projectAgentEvent({
        event: toAgentStreamEvent(event),
        runtime,
      }));
    }
    return eventRunState.historySync;
  }, [applyRuntimeActions]);
}

function useRecoverTurnDraft(options: RecoverTurnDraftOptions) {
  const {
    hydrateTurnDraft,
    recoveryRunRef,
    setActiveRun,
    setChatError,
    setLoading,
    setStopPending,
    stopRecoveredRun,
  } = options;

  return useCallback((sessionId: string, draft: SessionTurnDraft | null | undefined) => {
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
    startRecoveredSessionStream({
      ...options,
      abortController,
      runtime,
      sessionId,
    });
  }, [
    hydrateTurnDraft,
    options,
    recoveryRunRef,
    setActiveRun,
    setChatError,
    setLoading,
    setStopPending,
    stopRecoveredRun,
  ]);
}

function startRecoveredSessionStream(options: RecoveredStreamOptions): void {
  const { abortController, applyRecoveredEvent, runtime, sessionId, syncRecoveredTurn } = options;

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
    options.setChatError(sessionId, toErrorMessage(error, options.requestFailedText));
  });
}

function resolveRecoveredTurnError(
  draft: SessionTurnDraft | null | undefined,
  messageText?: string,
): string {
  return messageText ?? (draft?.status === 'error' ? (draft.error ?? '') : '');
}
