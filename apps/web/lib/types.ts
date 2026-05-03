// Web UI types plus API contracts generated from core/shared/schema.json.

import type {
  AgentMessageTaskCreateRequest as SharedAgentMessageTaskCreateRequest,
  AgentMessageTaskPayload as SharedAgentMessageTaskPayload,
  AskHumanOption,
  ProviderConfig,
  SessionFileContent as SharedSessionFileContent,
  SessionImageContent as SharedSessionImageContent,
  SessionMessage as SharedSessionMessage,
  TaskRuntimeOverrides as SharedTaskRuntimeOverrides,
  WorkflowDefinition as SharedWorkflowDefinition,
  WorkflowEdge as SharedWorkflowEdge,
  WorkflowInputVariable as SharedWorkflowInputVariable,
  WorkflowNode as SharedWorkflowNode,
  WorkflowTaskCreateRequest as SharedWorkflowTaskCreateRequest,
  WorkflowTaskPayload as SharedWorkflowTaskPayload,
  TaskPayload as SharedTaskPayload,
  TaskUpdateRequest as SharedTaskUpdateRequest,
} from '@/lib/envelope.generated';

export type {
  AgentIterationSummaryItem,
  AgentCompletionDeltaPayload,
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
  SessionSidebarPartition,
  SessionSidebarPartitionPutRequest,
  SessionSidebarPartitionState,
  SessionMessagePage,
  SessionToolCall,
  SessionToolResult,
  SetActiveProviderRequest,
} from '@/lib/envelope.generated';

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
export type TaskRuntimeOverrides = SharedTaskRuntimeOverrides;
export type WorkflowDefinition = SharedWorkflowDefinition;
export type WorkflowNode = SharedWorkflowNode;
export type WorkflowEdge = SharedWorkflowEdge;
export type WorkflowInputVariable = SharedWorkflowInputVariable;

export interface TaskRunNodeResult {
  node_id: string;
  node_type: string;
  status: string;
  started_at?: string;
  finished_at?: string;
  completed_seq?: number;
  branch_id?: string;
  input?: unknown;
  output?: unknown;
  preview?: string;
  error?: string;
}

export interface TaskRunLog {
  task_id: string;
  run_id: string;
  trace_id: string;
  task_kind?: 'agent_message' | 'workflow';
  action?: string;
  scheduled_at: string;
  started_at?: string;
  finished_at?: string;
  status: 'success' | 'cancelled' | 'error' | 'skipped' | 'awaiting_human';
  session_id_input?: string;
  session_id_output?: string;
  response_preview?: string;
  node_results?: TaskRunNodeResult[];
  error?: string;
}

export interface TextTaskCreateRequest extends Omit<SharedAgentMessageTaskCreateRequest, 'task_kind' | 'trace_id'> {
  task_kind: 'agent_message';
}

export interface WorkflowTaskCreateRequest extends Omit<
  SharedWorkflowTaskCreateRequest,
  'trace_id'
> {
  task_kind: 'workflow';
}

export type TaskCreateRequest = TextTaskCreateRequest | WorkflowTaskCreateRequest;

export interface TaskUpdateRequest extends Pick<
  SharedTaskUpdateRequest,
  | 'message'
  | 'session_id'
  | 'runtime_overrides'
  | 'interval_seconds'
  | 'cron_expr'
  | 'enabled'
  | 'task_kind'
  | 'workflow'
> {
}

export type TextTaskUpdateRequest = TaskUpdateRequest;
