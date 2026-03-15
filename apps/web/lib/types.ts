// Web UI types plus API contracts generated from core/shared/schema.json.

import type {
  AskHumanOption,
  ProviderConfig,
  SessionFileContent as SharedSessionFileContent,
  SessionMessage as SharedSessionMessage,
} from '@/lib/envelope.generated';

export type {
  AgentRequest,
  AgentSendAwaitingHumanResponse,
  AgentSendResponse,
  AgentStopResponsePayload,
  AgentSendSuccessResponse,
  ApiEnvelope,
  ApiErrorEnvelope,
  ApiRequest,
  ApiSuccessEnvelope,
  AskHumanOption,
  AssistantSessionEndSignal,
  BridgeConfig,
  ConfigUpdate,
  GraphQLDomainInput,
  GraphQLDomainResponse,
  GraphQLMutationPolicyInput,
  GraphQLMutationPolicyResponse,
  GraphQLSourceInput,
  GraphQLSourceResponse,
  HumanResponseAck,
  HumanResponseRequest,
  ProviderConfig,
  ProviderConfigInput,
  ProviderListResponse,
  SessionContentPart,
  SessionDetail,
  SessionFileContent,
  SessionImageContent,
  SessionHumanInteraction,
  SessionMessage,
  SessionMetadata,
  SessionToolCall,
  SessionToolResult,
  SetActiveProviderRequest,
} from '@/lib/envelope.generated';

export interface UserChatMessage {
  id: string;
  kind: 'user';
  content: string;
}

export interface AssistantChatMessage {
  id: string;
  kind: 'assistant';
  content: string;
}

export interface SystemChatMessage {
  id: string;
  kind: 'system';
  content: string;
}

export interface ToolChatMessage {
  id: string;
  kind: 'tool';
  content: string;
  attachments?: ChatFileAttachment[];
  toolName?: string;
  toolStatus?: string;
  toolCallId?: string;
  toolCalls?: unknown[];
  traceId?: string;
  rawOutput?: string;
}

export interface ChatFileAttachment {
  artifactId: string;
  name: string;
  downloadUrl: string;
  mimeType?: string;
  bytes?: number;
  sha256?: string;
  sourcePath?: string;
  note?: string;
}

export interface ErrorChatMessage {
  id: string;
  kind: 'error';
  content: string;
  error: string;
}

export interface PendingQuestionMessage {
  id: string;
  kind: 'pending_question';
  content: string;
  questionId: string;
  sessionId: string;
  selectionMode?: 'single' | 'multiple';
  options?: AskHumanOption[];
}

export interface QuestionChatMessage {
  id: string;
  kind: 'question';
  content: string;
  questionId?: string;
  selectionMode?: 'single' | 'multiple';
  options?: AskHumanOption[];
}

export type ChatMessage =
  | UserChatMessage
  | AssistantChatMessage
  | SystemChatMessage
  | ToolChatMessage
  | ErrorChatMessage
  | QuestionChatMessage
  | PendingQuestionMessage;

export interface ProviderModelOption {
  providerName: string;
  providerType: ProviderConfig['type'];
  model: string;
}

export type SessionMessageRole = NonNullable<SharedSessionMessage['role']>;
export type SessionFileAttachment = SharedSessionFileContent;

export interface RSSBriefingHighlight {
  rank: number;
  group_id: string;
  topic_label?: string;
  headline: string;
  summary?: string;
  why_it_matters?: string;
  importance: 'low' | 'normal' | 'high';
  source_item_count: number;
  source_feed_count: number;
  tags?: string[];
}

export interface RSSBriefing {
  id?: string;
  title: string;
  summary?: string;
  generated_at: string;
  saved_at?: string;
  window_hours: number;
  scanned_groups: number;
  highlight_count: number;
  trace_id?: string;
  task_id?: string;
  highlights: RSSBriefingHighlight[];
}
