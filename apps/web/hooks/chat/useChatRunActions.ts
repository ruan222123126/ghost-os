import { useCallback } from 'react';
import { stopAgent, stopExternalAgent } from '@/lib/api/agent/api';
import { createClientTraceId } from '@/lib/api/trace';
import { draftImagesToChatImages, draftImagesToSessionImages } from '@/lib/chatImageDrafts';
import { buildUserMessage } from '@/lib/chatMessages';
import { isAbortError, isAgentRunCancellationMessage, toErrorMessage } from '@/lib/errors';
import { buildAgentMessageWithSelectedSkill } from '@/lib/selectedSkillMessage';
import type { AgentRuntimeType, ChatSendInput, ExternalCodexPermissionMode } from '@/lib/types';
import type {
  ChatStateControls,
  ChatStreamRunResult,
  StreamAgentRunInput,
  UseBridgeChatOptions,
} from './types';

export interface UseChatRunActionsOptions {
  appendErrorMessage: ChatStateControls['appendErrorMessage'];
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  beginHistorySync: ChatStateControls['beginHistorySync'];
  clearChatError: ChatStateControls['clearChatError'];
  clearStreamingState: ChatStateControls['clearStreamingState'];
  codexImagesUnsupportedText: string;
  codexUnavailableText: string;
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  endHistorySync: ChatStateControls['endHistorySync'];
  externalCodexPermissionMode?: ExternalCodexPermissionMode;
  externalProjectRoot?: string;
  getCurrentSessionId: () => string;
  getActiveRun: ChatStateControls['getActiveRun'];
  getStopPending: ChatStateControls['getStopPending'];
  hasPendingQuestionInSession: ChatStateControls['hasPendingQuestionInSession'];
  markBackgroundCompleted: ChatStateControls['markBackgroundCompleted'];
  migrateSessionState: ChatStateControls['migrateSessionState'];
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  requestPostSendFocus: (messageId: string) => void;
  requestFailedText: string;
  runAgentStream: (run: StreamAgentRunInput) => Promise<ChatStreamRunResult>;
  resolveActiveRunSessionId: ChatStateControls['resolveActiveRunSessionId'];
  setActiveRun: ChatStateControls['setActiveRun'];
  setLoading: ChatStateControls['setLoading'];
  setStopPending: ChatStateControls['setStopPending'];
  setChatError: ChatStateControls['setChatError'];
  syncRecentHistory: (sessionId: string) => Promise<void>;
}

interface UseStopCurrentRunOptions extends UseChatRunActionsOptions {
  syncStoppedRunHistory: (sessionId: string) => Promise<void>;
}

interface AgentRunContext {
  abortController: AbortController;
  agentMessage: string;
  runtime: AgentRuntimeType;
  sessionId: string;
  traceId: string;
}

interface StopRunResult {
  stopAccepted: boolean;
  targetSessionId: string;
}

export function useChatRunActions(options: UseChatRunActionsOptions) {
  const syncStoppedRunHistory = useStoppedRunHistorySync(options);

  return {
    sendChatMessage: useSendChatMessage(options),
    stopCurrentRun: useStopCurrentRun({
      ...options,
      syncStoppedRunHistory,
    }),
  };
}

function useStoppedRunHistorySync(options: UseChatRunActionsOptions) {
  const {
    beginHistorySync,
    endHistorySync,
    getCurrentSessionId,
    onSessionResolved,
    requestFailedText,
    setChatError,
    syncRecentHistory,
  } = options;

  return useCallback(async (sessionId: string) => {
    const trimmedSessionId = sessionId.trim();
    if (!trimmedSessionId) {
      return;
    }
    if (trimmedSessionId !== getCurrentSessionId().trim()) {
      onSessionResolved?.(trimmedSessionId);
    }

    beginHistorySync(trimmedSessionId);
    try {
      await syncRecentHistory(trimmedSessionId);
    } catch (error) {
      setChatError(trimmedSessionId, toErrorMessage(error, requestFailedText));
    } finally {
      endHistorySync(trimmedSessionId);
    }
  }, [
    beginHistorySync,
    endHistorySync,
    getCurrentSessionId,
    onSessionResolved,
    requestFailedText,
    setChatError,
    syncRecentHistory,
  ]);
}

function useSendChatMessage(options: UseChatRunActionsOptions) {
  const { runAgentStream } = options;

  return useCallback(async (input: ChatSendInput) => {
    if (!hasSendPayload(input)) {
      return;
    }

    const context = beginAgentRun(input, options);
    try {
      const result = await runAgentStream(buildStreamRunInput(input, context, options));
      markCompletedRunIfNeeded(result, context, options);
    } catch (error) {
      handleAgentRunError(error, context, options);
    } finally {
      finishAgentRun(context, options);
    }
  }, [options, runAgentStream]);
}

function useStopCurrentRun(options: UseStopCurrentRunOptions) {
  const { currentSessionId, getActiveRun, getStopPending } = options;

  return useCallback(async () => {
    const sessionId = currentSessionId.trim();
    const run = getActiveRun(sessionId);
    if (!run || getStopPending(sessionId)) {
      return;
    }

    options.clearChatError(sessionId);
    options.setStopPending(sessionId, true);
    let stopRunResult: StopRunResult = { stopAccepted: false, targetSessionId: sessionId };
    try {
      stopRunResult = await stopActiveAgentRun(run, options);
    } catch (error) {
      options.setChatError(stopRunResult.targetSessionId, toErrorMessage(error, options.requestFailedText));
    } finally {
      finishStoppedRun(stopRunResult, options);
    }
  }, [currentSessionId, getActiveRun, getStopPending, options]);
}

function beginAgentRun(input: ChatSendInput, options: UseChatRunActionsOptions): AgentRunContext {
  const sessionId = options.currentSessionId.trim();
  const traceId = createClientTraceId('agent-run');
  const userMessageId = `local:user:${traceId}`;
  const abortController = new AbortController();
  const runtime = resolveAgentRuntime(input.agentRuntime);
  options.clearChatError(sessionId);
  options.clearStreamingState(sessionId);
  options.setStopPending(sessionId, false);
  options.setActiveRun(sessionId, { abortController, runtime, sessionId, traceId });
  options.appendCommittedMessages(sessionId, [buildUserMessage(input.message, {
    id: userMessageId,
    images: draftImagesToChatImages(input.images),
    selectedSkill: input.selectedSkill,
  })]);
  options.requestPostSendFocus(userMessageId);
  options.setLoading(sessionId, true);

  return {
    abortController,
    agentMessage: buildAgentMessageWithSelectedSkill(input),
    runtime,
    sessionId,
    traceId,
  };
}

function buildStreamRunInput(
  input: ChatSendInput,
  context: AgentRunContext,
  options: UseChatRunActionsOptions,
): StreamAgentRunInput {
  if (context.runtime === 'codex') {
    if (input.images.length > 0) {
      throw new Error(options.codexImagesUnsupportedText);
    }
    if (!options.externalCodexPermissionMode) {
      throw new Error(options.codexUnavailableText);
    }

    return {
      agentRuntime: 'codex',
      codexMode: input.codexMode,
      message: context.agentMessage,
      model: input.model?.trim() || undefined,
      permissionMode: options.externalCodexPermissionMode,
      projectRoot: options.externalProjectRoot,
      sessionId: context.sessionId || undefined,
      signal: context.abortController.signal,
      traceId: context.traceId,
    };
  }

  return {
    agentRuntime: 'ghost',
    images: draftImagesToSessionImages(input.images),
    message: context.agentMessage,
    mode: input.mode,
    sessionId: context.sessionId || undefined,
    signal: context.abortController.signal,
    traceId: context.traceId,
  };
}

function markCompletedRunIfNeeded(
  result: ChatStreamRunResult,
  context: AgentRunContext,
  options: UseChatRunActionsOptions,
): void {
  if (shouldMarkBackgroundCompleted({
    currentSessionId: options.getCurrentSessionId(),
    hasPendingQuestion: options.hasPendingQuestionInSession(result.sessionId),
    initialSessionId: context.sessionId,
    result,
  })) {
    options.markBackgroundCompleted(result.sessionId);
  }
}

function handleAgentRunError(error: unknown, context: AgentRunContext, options: UseChatRunActionsOptions): void {
  const targetSessionId = options.resolveActiveRunSessionId(context.traceId, context.sessionId);
  if (!shouldSuppressRunError(error, options.getStopPending(targetSessionId), context.abortController.signal.aborted)) {
    options.appendErrorMessage(targetSessionId, toErrorMessage(error, options.requestFailedText));
  }
}

function finishAgentRun(context: AgentRunContext, options: UseChatRunActionsOptions): void {
  const targetSessionId = options.resolveActiveRunSessionId(context.traceId, context.sessionId);
  options.setLoading(targetSessionId, false);
  options.setActiveRun(targetSessionId, null);
  options.setStopPending(targetSessionId, false);
}

async function stopActiveAgentRun(
  run: NonNullable<ReturnType<ChatStateControls['getActiveRun']>>,
  options: UseStopCurrentRunOptions,
): Promise<StopRunResult> {
  const response = run.runtime === 'codex'
    ? await stopExternalAgent(run.sessionId)
    : await stopAgent(run.sessionId || undefined, run.traceId || undefined);
  run.abortController?.abort();
  const targetSessionId = resolveStopSessionId(response.session_id, run.sessionId);
  if (targetSessionId && targetSessionId !== run.sessionId) {
    options.migrateSessionState(run.sessionId, targetSessionId);
  }
  await options.syncStoppedRunHistory(targetSessionId);
  return { stopAccepted: true, targetSessionId };
}

function finishStoppedRun(result: StopRunResult, options: UseStopCurrentRunOptions): void {
  if (result.stopAccepted) {
    options.setLoading(result.targetSessionId, false);
    options.setActiveRun(result.targetSessionId, null);
  }
  options.setStopPending(result.targetSessionId, false);
}

function hasSendPayload(input: ChatSendInput): boolean {
  return input.message.trim().length > 0 || input.images.length > 0 || input.selectedSkill !== undefined;
}

function shouldSuppressRunError(error: unknown, stopPending: boolean, streamAborted: boolean): boolean {
  if (isAbortError(error)) {
    return true;
  }

  const message = toErrorMessage(error);
  if (!stopPending && !streamAborted) {
    return false;
  }

  return isAgentRunCancellationMessage(message);
}

function shouldMarkBackgroundCompleted(input: {
  currentSessionId: string;
  hasPendingQuestion: boolean;
  initialSessionId: string;
  result: ChatStreamRunResult;
}): boolean {
  const completedSessionId = input.result.sessionId.trim();
  if (!completedSessionId || input.result.terminalType !== 'done' || input.hasPendingQuestion) {
    return false;
  }

  const currentSessionId = input.currentSessionId.trim();
  if (currentSessionId === completedSessionId) {
    return false;
  }
  return input.initialSessionId.trim().length > 0 || currentSessionId.length > 0;
}

function resolveStopSessionId(stoppedSessionId?: string, activeSessionId?: string): string {
  const trimmedStoppedSessionId = stoppedSessionId?.trim();
  if (trimmedStoppedSessionId) {
    return trimmedStoppedSessionId;
  }
  return activeSessionId?.trim() ?? '';
}

function resolveAgentRuntime(runtime?: AgentRuntimeType): AgentRuntimeType {
  return runtime === 'codex' ? 'codex' : 'ghost';
}
