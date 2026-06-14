// Web UI types plus API contracts generated from core/shared/schema.json.

import type {
  AgentMessageTaskCreateRequest as SharedAgentMessageTaskCreateRequest,
  AgentMessageTaskPayload as SharedAgentMessageTaskPayload,
  OrchestrationAgentNode as SharedOrchestrationAgentNode,
  OrchestrationDefinition as SharedOrchestrationDefinition,
  OrchestrationEdge as SharedOrchestrationEdge,
  OrchestrationGroupNode as SharedOrchestrationGroupNode,
  OrchestrationNode as SharedOrchestrationNode,
  OrchestrationTaskCreateRequest as SharedOrchestrationTaskCreateRequest,
  OrchestrationTaskPayload as SharedOrchestrationTaskPayload,
  AskHumanOption,
  ProviderConfig,
  SessionImageContent as SharedSessionImageContent,
  SessionMessage as SharedSessionMessage,
  SessionTurnDraft as SharedSessionTurnDraft,
  TaskRelayConfig as SharedTaskRelayConfig,
  TaskRuntimeOverrides as SharedTaskRuntimeOverrides,
  WorkflowAgentNode as SharedWorkflowAgentNode,
  WorkflowDefinition as SharedWorkflowDefinition,
  WorkflowEdge as SharedWorkflowEdge,
  WorkflowIfNode as SharedWorkflowIfNode,
  WorkflowInputVariable as SharedWorkflowInputVariable,
  WorkflowLLMNode as SharedWorkflowLLMNode,
  WorkflowLoopNode as SharedWorkflowLoopNode,
  WorkflowNode as SharedWorkflowNode,
  WorkflowStartNode as SharedWorkflowStartNode,
  TaskPatchRequest as SharedTaskPatchRequest,
  WorkflowTaskCreateRequest as SharedWorkflowTaskCreateRequest,
  WorkflowTaskPayload as SharedWorkflowTaskPayload,
  WorkflowToolNode as SharedWorkflowToolNode,
  TaskPayload as SharedTaskPayload,
} from '@/lib/envelope.generated';

export type {
  AgentCompletionDeltaPayload,
  AgentAwaitingHumanStreamPayload,
  AgentDonePayload,
  AgentErrorPayload,
  AgentRequest,
  AgentSendAwaitingHumanResponse,
  AgentSendResponse,
  AgentStopResponsePayload,
  AgentSendSuccessResponse,
  AgentStreamEvent,
  AgentStreamMessagePayload,
  AgentToolCallFinishedPayload,
  AgentToolCallStartedPayload,
  ApiEnvelope,
  ApiErrorEnvelope,
  ApiRequest,
  ApiSuccessEnvelope,
  AskHumanOption,
  AssistantSessionEndSignal,
  AgentRunStartedPayload,
  BridgeConfig,
  ConfigUpdate,
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
  SessionPushAssistantMessagePayload,
  SessionPushAwaitingHumanPayload,
  TaskRunCardEventPayload,
  TaskRunCardFinishedPayload,
  TaskRunCardStartedPayload,
  TaskRunLog,
  TaskRunNodeResult,
  TaskRunPayload,
  TaskRunStopRequest,
  TaskRunStopResponse,
  SessionPushEvent,
  SessionSourceAssignment,
  SessionSourceResolution,
  SessionSidebarPartition,
  SessionSidebarPartitionPutRequest,
  SessionSidebarPartitionState,
  SessionMessagePage,
  SessionToolCall,
  SessionToolResult,
  SetActiveProviderRequest,
} from '@/lib/envelope.generated';

export type { TaskRunCard } from '@/lib/taskRunCards';
export type SessionTurnDraft = SharedSessionTurnDraft;

export interface UserChatMessage {
  id: string;
  kind: 'user';
  content: string;
  images?: ChatImage[];
}

export interface AssistantChatMessage {
  id: string;
  kind: 'assistant';
  content: string;
  inProgress?: boolean;
}

export interface ThinkingChatMessage {
  id: string;
  kind: 'thinking';
  content: string;
  inProgress?: boolean;
}

export interface EventChatMessage {
  id: string;
  kind: 'event';
  content: string;
}

export type SystemChatMessageSourceRole = 'system' | 'internal';

export interface SystemChatMessage {
  id: string;
  kind: 'system';
  content: string;
  sourceRole?: SystemChatMessageSourceRole;
}

export interface ToolChatMessage {
  id: string;
  kind: 'tool';
  content: string;
  images?: ChatImage[];
  attachments?: ChatFileAttachment[];
  toolInput?: string;
  toolName?: string;
  toolStatus?: string;
  toolCallId?: string;
  toolCalls?: unknown[];
  traceId?: string;
  rawOutput?: string;
}

export interface StreamingToolState {
  id: string;
  content: string;
  toolInput?: string;
  toolName?: string;
  toolStatus?: string;
  toolCallId?: string;
  traceId?: string;
}

export interface StreamingAssistantSegment {
  id: string;
  content: string;
}

export interface StreamingThinkingSegment {
  id: string;
  content: string;
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

export interface ChatImage {
  id: string;
  name?: string;
  url?: string;
  path?: string;
  mimeType?: string;
  width?: number;
  height?: number;
  bytes?: number;
  sha256?: string;
}

export interface ChatImageDraft {
  id: string;
  name: string;
  content: SharedSessionImageContent;
}

export interface ChatSendInput {
  message: string;
  images: ChatImageDraft[];
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
  | ThinkingChatMessage
  | EventChatMessage
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

export type SkillSource = 'repo' | 'user';

export interface SkillPayload {
  id: string;
  name: string;
  description: string;
  path: string;
  source: SkillSource;
  enabled: boolean;
}

export interface SkillUpdateRequest {
  enabled: boolean;
  trace_id?: string;
}

export interface ToolPayload {
  name: string;
  enabled: boolean;
  prompt_override?: string;
  sandbox_memory_mb?: number;
  input_schema?: Record<string, unknown>;
}

export interface ToolUpdateRequest {
  enabled?: boolean;
  prompt_override?: string;
  sandbox_memory_mb?: number;
  trace_id?: string;
}

export type PromptInsertPoint = 'rule' | 'core_job' | 'memory' | 'context';

export interface PromptLibraryItem {
  id: string;
  name: string;
  insert_point: PromptInsertPoint;
  content: string;
  active: boolean;
}

export interface SystemPromptToolDefinition {
  name: string;
  description: string;
  parameters?: Record<string, unknown>;
}

export interface SystemPromptPayload {
  core_prompt: string;
  rendered_prompt: string;
  prompt_library: PromptLibraryItem[];
  tool_definitions: SystemPromptToolDefinition[];
}

export interface SystemPromptUpdateRequest {
  core_prompt?: string;
  prompt_library?: PromptLibraryItem[];
  trace_id?: string;
}

export interface PresetPromptRefs {
  rule?: string;
  core_job?: string;
  memory?: string;
  context?: string[];
}

export interface PresetPayload {
  id: string;
  name: string;
  tool_allowlist: string[];
  prompt_refs: PresetPromptRefs;
}

export interface PresetCreateRequest {
  name: string;
  tool_allowlist?: string[];
  prompt_refs?: PresetPromptRefs;
  trace_id?: string;
}

export interface PresetUpdateRequest {
  name?: string;
  tool_allowlist?: string[];
  prompt_refs?: PresetPromptRefs;
  trace_id?: string;
}

export type TaskPayload = SharedTaskPayload;
export type AgentMessageTaskPayload = SharedAgentMessageTaskPayload;
export type WorkflowTaskPayload = SharedWorkflowTaskPayload;
export type OrchestrationTaskPayload = SharedOrchestrationTaskPayload;
export type TaskRuntimeOverrides = SharedTaskRuntimeOverrides;
export type TaskRelayConfig = SharedTaskRelayConfig;
export type WorkflowDefinition = SharedWorkflowDefinition;
export type WorkflowNode = SharedWorkflowNode;
export type WorkflowEdge = SharedWorkflowEdge;
export type WorkflowStartNode = SharedWorkflowStartNode;
export type WorkflowInputVariable = SharedWorkflowInputVariable;
export type WorkflowToolNode = SharedWorkflowToolNode;
export type WorkflowLLMNode = SharedWorkflowLLMNode;
export type WorkflowAgentNode = SharedWorkflowAgentNode;
export type WorkflowIfNode = SharedWorkflowIfNode;
export type WorkflowLoopNode = SharedWorkflowLoopNode;
export type OrchestrationDefinition = SharedOrchestrationDefinition;
export type OrchestrationNode = SharedOrchestrationNode;
export type OrchestrationEdge = SharedOrchestrationEdge;
export type OrchestrationGroupNode = SharedOrchestrationGroupNode;
export type OrchestrationAgentNode = SharedOrchestrationAgentNode;

export type TextTaskCreateRequest = SharedAgentMessageTaskCreateRequest & { task_kind: 'agent_message' };
export type WorkflowTaskCreateRequest = SharedWorkflowTaskCreateRequest;
export type OrchestrationTaskCreateRequest = SharedOrchestrationTaskCreateRequest;

export type TaskPatchRequest = SharedTaskPatchRequest;
export type TaskUpdateRequest = TaskPatchRequest;

export type TextTaskUpdateRequest = TaskUpdateRequest;
