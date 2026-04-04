import type { ChatStateControls, UseBridgeChatOptions } from './types';
import type { ToolTagStreamState } from '@/lib/toolTagText';

export const TOOL_PENDING_STATUS = 'pending';
export const TOOL_RUNNING_STATUS = 'running';
export const TOOL_SUCCESS_STATUS = 'success';
export const TOOL_ERROR_STATUS = 'error';

export interface StreamRuntimeState {
  assistantBuffer: string;
  assistantMessageId: string;
  pendingPreviewQueue: string[];
  previewToolArgs: Map<string, string>;
  previewToolCallSeqToID: Map<number, string>;
  previewToolIDByMessageId: Map<string, string>;
  sessionId: string;
  toolMessageIds: Map<string, string>;
  toolTagState: ToolTagStreamState;
  traceId: string;
}

export interface StreamHumanRunOptions {
  answer: string;
  cancelled?: boolean;
  questionId: string;
  sessionId: string;
  signal?: AbortSignal;
  traceId: string;
}

export interface UseChatStreamControllerOptions {
  activeRunRef: ChatStateControls['activeRunRef'];
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'];
  clearStreamingAssistantText: ChatStateControls['clearStreamingAssistantText'];
  clearStreamingState: ChatStateControls['clearStreamingState'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  setActiveRun: ChatStateControls['setActiveRun'];
  syncRecentHistory: (sessionId: string) => Promise<void>;
  upsertPendingQuestion: ChatStateControls['upsertPendingQuestion'];
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'];
}

export interface EventApplyOptions {
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'];
  clearStreamingAssistantText: ChatStateControls['clearStreamingAssistantText'];
  state: StreamRuntimeState;
  upsertPendingQuestion: ChatStateControls['upsertPendingQuestion'];
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'];
}
