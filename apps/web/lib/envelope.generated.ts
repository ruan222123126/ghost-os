// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

export type BusAction = 'AGENT_SEND' | 'AGENT_STOP' | 'HUMAN_RESPONSE' | 'SESSIONS_LIST' | 'SESSIONS_SEARCH' | 'SESSION_GET' | 'CONFIG_GET' | 'CONFIG_UPDATE' | 'CONFIG_PROVIDERS_GET' | 'CONFIG_PROVIDER_CREATE' | 'CONFIG_PROVIDER_UPDATE' | 'CONFIG_PROVIDER_DELETE' | 'SKILL_LIST' | 'SKILL_UPDATE' | 'SKILL_DELETE' | 'TASK_CREATE' | 'TASK_LIST' | 'TASK_GET' | 'TASK_UPDATE' | 'TASK_RUN_NOW' | 'TASK_STOP' | 'TASK_LOGS' | 'TASK_DELETE';
export type BusStatus = 'success' | 'error';

export interface ApiRequest<TParams extends object> {
  action: BusAction;
  params: TParams;
  trace_id: string;
  request_id?: string;
}

export interface ApiSuccessEnvelope<TPayload> {
  status: 'success';
  payload: TPayload;
  error: string;
  request_id?: string;
}

export interface ApiErrorEnvelope {
  status: 'error';
  payload: Record<string, unknown>;
  error: string;
  request_id?: string;
}

export type ApiEnvelope<TPayload> = ApiSuccessEnvelope<TPayload> | ApiErrorEnvelope;

export interface AssistantSessionEndSignal {
  signal: 'END_SESSION';
  message: string;
}

export interface AgentRequest {
  mode?: 'plan';
  message?: string;
  images?: SessionImageContent[];
  session_id?: string;
  project_root?: string;
  trace_id?: string;
}

export interface AskHumanOption {
  label: string;
  allow_custom?: boolean;
}

export interface AgentSendSuccessResponse {
  message: string;
  session_id: string;
  session_ended: boolean;
  mode?: 'plan';
  session_end?: AssistantSessionEndSignal | null;
}

export interface AgentSendAwaitingHumanResponse {
  status: 'awaiting_human';
  session_id: string;
  question_id: string;
  prompt: string;
  selection_mode?: 'single' | 'multiple';
  options?: AskHumanOption[];
}

export interface AgentStopRequest {
  session_id?: string;
  trace_id?: string;
}

export interface AgentStopResponsePayload {
  status: 'stopped' | 'not_running';
  message: string;
  session_id?: string;
}

export interface HumanResponseRequest {
  session_id: string;
  question_id: string;
  answer: string;
  cancelled?: boolean;
}

export interface HumanResponseAck {
  session_id: string;
  question_id: string;
  accepted: boolean;
}

export interface AgentStreamEvent {
  id: string;
  step_id: string;
  trace_id: string;
  session_id?: string;
  turn: number;
  type: 'run_started' | 'completion_delta' | 'tool_call_started' | 'tool_call_finished' | 'awaiting_human' | 'message' | 'done' | 'error';
  payload: Record<string, unknown>;
  at?: string;
}

export interface SessionImageContent {
  path?: string;
  url?: string;
  mime_type?: string;
  width?: number;
  height?: number;
  sha256?: string;
  bytes?: number;
}

export interface SessionFileContent {
  artifact_id: string;
  name: string;
  mime_type?: string;
  bytes?: number;
  sha256?: string;
  download_url: string;
  source_path?: string;
  note?: string;
}

export interface SessionPushEvent {
  id: string;
  type: 'assistant_message' | 'awaiting_human' | 'run_started' | 'completion_delta' | 'tool_call_started' | 'tool_call_finished' | 'error' | 'done' | 'task_run_card_started' | 'task_run_card_event' | 'task_run_card_finished';
  trace_id?: string;
  session_id: string;
  payload: Record<string, unknown>;
  at?: string;
}

export interface AgentRunStartedPayload {
  session_id?: string;
}

export interface SessionContentPart {
  type: string;
  text?: string;
  image?: SessionImageContent | null;
  file?: SessionFileContent | null;
}

export interface AgentCompletionDeltaPayload {
  kind: 'text' | 'thinking' | 'tool_call_start' | 'tool_call_delta' | 'tool_call_end';
  text?: string;
  thinking?: string;
  tool_call_index?: number;
  tool_call_id?: string;
  tool_name?: string;
  arguments_fragment?: string;
}

export interface SessionToolCall {
  id: string;
  name: string;
  arguments: Record<string, unknown>;
}

export interface AgentToolCallStartedPayload {
  tool?: string;
  tool_call_id?: string;
  arguments_json?: string;
}

export interface SessionToolResult {
  status: 'success' | 'error';
  tool: string;
  trace_id?: string;
  output?: string;
  error?: string;
}

export interface AgentToolCallFinishedPayload {
  tool?: string;
  tool_call_id?: string;
  status?: string;
  error?: string;
  output?: string;
}

export interface SessionHumanInteraction {
  question_id: string;
  prompt: string;
  selection_mode?: 'single' | 'multiple';
  options?: AskHumanOption[];
  answer?: string;
}

export interface AgentAwaitingHumanStreamPayload {
  tool?: string;
  tool_call_id?: string;
  question_id: string;
  prompt: string;
  selection_mode?: 'single' | 'multiple';
  options?: AskHumanOption[];
}

export interface AgentStreamMessagePayload {
  text: string;
  session_id?: string;
}

export interface SessionMessage {
  index: number;
  role: 'system' | 'internal' | 'user' | 'assistant' | 'tool';
  text?: string;
  content?: SessionContentPart[];
  tool_calls?: SessionToolCall[];
  tool_result?: SessionToolResult | null;
  human_interaction?: SessionHumanInteraction | null;
  tool_call_id?: string;
  in_progress?: boolean;
  thinking?: string;
}

export interface AgentDonePayload {
  session_id?: string;
  session_ended?: boolean;
}

export interface SessionMetadata {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
  message_count: number;
  token_count: number;
}

export interface SessionSidebarPartition {
  id: string;
  name: string;
}

export interface SessionSidebarPartitionState {
  version: number;
  partitions: SessionSidebarPartition[];
  assignments: Record<string, string>;
}

export interface SessionSidebarPartitionPutRequest {
  version: number;
  partitions: SessionSidebarPartition[];
  assignments: Record<string, string>;
  trace_id?: string;
}

export interface SessionSourceAssignment {
  kind: 'workflow' | 'orchestration' | 'loop' | 'task';
  owner_id: string;
  owner_name: string;
}

export interface SessionSourceResolution {
  assignments: Record<string, SessionSourceAssignment>;
  hidden_session_ids: string[];
}

export interface SessionMessagePage {
  limit: number;
  before?: number | null;
  start_index?: number | null;
  end_index?: number | null;
  has_more_before: boolean;
  next_before?: number | null;
}

export interface SessionTurnDraftSegment {
  id: string;
  content: string;
}

export interface SessionTurnDraftTool {
  id: string;
  content: string;
  tool_input?: string;
  tool_name?: string;
  tool_status?: string;
  tool_call_id?: string;
  trace_id?: string;
}

export interface AgentErrorPayload {
  message: string;
  session_id?: string;
  code?: number;
}

export interface SessionTurnDraftPendingQuestion {
  question_id: string;
  prompt: string;
  selection_mode?: 'single' | 'multiple';
  options?: AskHumanOption[];
}

export interface SessionDetail {
  id: string;
  title: string;
  messages: SessionMessage[];
  created_at: string;
  updated_at: string;
  message_count: number;
  page: SessionMessagePage;
  token_count: number;
  turn_draft?: SessionTurnDraft | null;
}

export interface SessionTurnDraft {
  trace_id: string;
  turn: number;
  status: 'streaming' | 'awaiting_human' | 'error';
  error?: string;
  pending_questions: SessionTurnDraftPendingQuestion[];
  assistant_segments: SessionTurnDraftSegment[];
  thinking_segments: SessionTurnDraftSegment[];
  tools: SessionTurnDraftTool[];
  item_order: string[];
}

export interface BridgeConfig {
  provider: string;
  provider_type: 'openai' | 'anthropic' | 'custom' | 'codex';
  base_url: string;
  model: string;
  chat_path: string;
  project_root: string;
  max_turns: number;
  task_execution_timeout_ms: number;
  relay_default_stop_policy: 'ai_decides' | 'max_rounds';
  relay_default_max_rounds: number;
  relay_default_execution_timeout_ms: number;
  llm_completion_retry_count: number;
  llm_completion_retry_interval_ms: number;
  api_key_set: boolean;
  model_selection_enabled: boolean;
  session_human_log_full_enabled: boolean;
  session_system_prompt_visible_enabled: boolean;
  assistant_markdown_enabled: boolean;
  tool_call_compact_output_enabled: boolean;
  memory_mode_enabled: boolean;
  microcompact_enabled: boolean;
  session_title_mode: 'session_id' | 'first_message' | 'ai_generated';
  web_search_tavily_url: string;
  web_search_exa_url: string;
  web_search_tavily_api_key_set: boolean;
  web_search_exa_api_key_set: boolean;
}

export interface SessionPushAssistantMessagePayload {
  message: string;
  session_ended: boolean;
  session_end?: AssistantSessionEndSignal | null;
}

export interface ConfigUpdate {
  provider?: string;
  api_key?: string;
  base_url?: string;
  model?: string;
  chat_path?: string;
  project_root?: string;
  max_turns?: number;
  task_execution_timeout_ms?: number;
  relay_default_stop_policy?: 'ai_decides' | 'max_rounds';
  relay_default_max_rounds?: number;
  relay_default_execution_timeout_ms?: number;
  llm_completion_retry_count?: number;
  llm_completion_retry_interval_ms?: number;
  session_human_log_full_enabled?: boolean;
  session_system_prompt_visible_enabled?: boolean;
  assistant_markdown_enabled?: boolean;
  tool_call_compact_output_enabled?: boolean;
  memory_mode_enabled?: boolean;
  microcompact_enabled?: boolean;
  session_title_mode?: 'session_id' | 'first_message' | 'ai_generated';
  web_search_tavily_url?: string;
  web_search_exa_url?: string;
  web_search_tavily_api_key?: string;
  web_search_exa_api_key?: string;
  trace_id?: string;
}

export interface SessionPushAwaitingHumanPayload {
  question_id: string;
  prompt: string;
  selection_mode?: 'single' | 'multiple';
  options?: AskHumanOption[];
}

export interface ProviderConfig {
  name: string;
  type: 'openai' | 'anthropic' | 'custom' | 'codex';
  base_url: string;
  models?: string[];
  context_window_tokens?: number;
  response_reserve_tokens?: number;
  model_context_window_tokens?: Record<string, number>;
  model_response_reserve_tokens?: Record<string, number>;
  api_key_set: boolean;
}

export interface TaskRunCardStartedPayload {
  card_id: string;
  run_id: string;
  kind: string;
  title?: string;
  node_id?: string;
  node_type?: string;
  round?: number;
  iteration?: number;
  branch_id?: string;
  source_session_id?: string;
  started_at: string;
}

export interface ProviderConfigInput {
  name: string;
  type: 'openai' | 'anthropic' | 'custom' | 'codex';
  base_url?: string;
  api_key?: string;
  models?: string[];
  context_window_tokens?: number;
  response_reserve_tokens?: number;
  model_context_window_tokens?: Record<string, number>;
  model_response_reserve_tokens?: Record<string, number>;
  trace_id?: string;
}

export interface TaskRunCardEventPayload {
  card_id: string;
  source_session_id?: string;
  source_event: AgentStreamEvent;
}

export interface ProviderListResponse {
  providers: ProviderConfig[];
  active_provider: string;
}

export interface TaskRunCardFinishedPayload {
  card_id: string;
  status: string;
  finished_at: string;
  preview?: string;
  error?: string;
  final_text?: string;
  source_session_id?: string;
}

export interface ProviderBusUpdateRequest {
  name: string;
  provider: ProviderConfigInput;
  trace_id?: string;
}

export interface SetActiveProviderRequest {
  name: string;
  trace_id?: string;
}

export interface WorkflowNode {
  id: string;
  type: 'start' | 'tool' | 'llm' | 'agent' | 'if' | 'loop' | 'end';
  start?: WorkflowStartNode;
  tool?: WorkflowToolNode;
  llm?: WorkflowLLMNode;
  agent?: WorkflowAgentNode;
  if?: WorkflowIfNode;
  loop?: WorkflowLoopNode;
}

export interface WorkflowEdge {
  from_node_id: string;
  to_node_id: string;
}

export interface WorkflowDefinition {
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
}

export interface OrchestrationNode {
  id: string;
  type: 'group' | 'agent';
  group?: OrchestrationGroupNode;
  agent?: OrchestrationAgentNode;
}

export interface OrchestrationGroupNode {
  title: string;
  shared_context: string;
  speaking_mode: 'sequential' | 'parallel' | 'owner';
  owner_agent_id?: string;
  max_rounds: number;
}

export interface AgentMessageTaskCreateRequest {
  message: string;
  session_id?: string;
  runtime_overrides?: TaskRuntimeOverrides;
  agent_mode?: 'single' | 'relay';
  relay?: TaskRelayConfig;
  task_kind?: 'agent_message';
  interval_seconds?: number;
  cron_expr?: string;
  trace_id?: string;
  scope?: 'user' | 'system' | 'orchestration';
}

export interface OrchestrationAgentNode {
  title: string;
  message: string;
  runtime_overrides?: TaskRuntimeOverrides;
}

export interface TaskRuntimeOverrides {
  provider_name?: string;
  model?: string;
  system_prompt?: string;
  preset_id?: string;
  tool_allowlist?: string[];
  tool_allowlist_only?: boolean;
  max_turns?: number;
}

export interface WorkflowToolNode {
  tool_name: string;
  arguments?: Record<string, unknown>;
}

export interface OrchestrationEdge {
  from_node_id: string;
  to_node_id: string;
  kind: 'control' | 'member';
}

export interface TaskRelayConfig {
  stop_policy: 'ai_decides' | 'max_rounds';
  max_rounds: number;
  execution_timeout_ms?: number;
}

export interface WorkflowLLMNode {
  prompt: string;
  system_prompt?: string;
}

export interface OrchestrationDefinition {
  nodes: OrchestrationNode[];
  edges: OrchestrationEdge[];
}

export interface WorkflowAgentNode {
  message: string;
  runtime_overrides?: TaskRuntimeOverrides;
}

export interface WorkflowIfNode {
  source_node_id?: string;
  operator: 'equals' | 'not_equals' | 'contains' | 'not_contains' | 'is_empty' | 'not_empty';
  value?: string;
  true_node_id: string;
  false_node_id: string;
}

export interface WorkflowTaskCreateRequest {
  task_kind: 'workflow';
  workflow: WorkflowDefinition;
  interval_seconds?: number;
  cron_expr?: string;
  trace_id?: string;
  scope?: 'user' | 'system' | 'orchestration';
}

export interface OrchestrationTaskCreateRequest {
  task_kind: 'orchestration';
  name: string;
  orchestration: OrchestrationDefinition;
  interval_seconds?: number;
  cron_expr?: string;
  trace_id?: string;
  scope?: 'user' | 'system' | 'orchestration';
}

export interface WorkflowLoopNode {
  max_iterations: number;
  body_node_id: string;
  exit_node_id: string;
}

export interface ProviderBusDeleteRequest {
  name: string;
  trace_id?: string;
}

export interface AgentMessageTaskPayload {
  id: string;
  message: string;
  session_id?: string;
  runtime_overrides?: TaskRuntimeOverrides;
  agent_mode?: 'single' | 'relay';
  relay?: TaskRelayConfig;
  task_kind: 'agent_message';
  schedule_type: 'interval' | 'cron';
  interval_seconds?: number;
  cron_expr?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  last_run_at?: string;
  next_run_at?: string;
  last_error?: string;
}

export interface TaskUpdateRequest {
  id: string;
  message?: string;
  name?: string;
  session_id?: string;
  runtime_overrides?: TaskRuntimeOverrides;
  agent_mode?: 'single' | 'relay';
  relay?: TaskRelayConfig;
  task_kind?: 'agent_message' | 'workflow' | 'orchestration';
  workflow?: WorkflowDefinition;
  orchestration?: OrchestrationDefinition;
  interval_seconds?: number;
  cron_expr?: string;
  enabled?: boolean;
  trace_id?: string;
  scope?: 'user' | 'system' | 'orchestration';
}

export interface TaskPatchRequest {
  message?: string;
  name?: string;
  session_id?: string;
  runtime_overrides?: TaskRuntimeOverrides;
  agent_mode?: 'single' | 'relay';
  relay?: TaskRelayConfig;
  task_kind?: 'agent_message' | 'workflow' | 'orchestration';
  workflow?: WorkflowDefinition;
  orchestration?: OrchestrationDefinition;
  interval_seconds?: number;
  cron_expr?: string;
  enabled?: boolean;
  trace_id?: string;
}

export interface WorkflowTaskPayload {
  id: string;
  task_kind: 'workflow';
  workflow: WorkflowDefinition;
  schedule_type: 'interval' | 'cron';
  interval_seconds?: number;
  cron_expr?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  last_run_at?: string;
  next_run_at?: string;
  last_error?: string;
}

export interface OrchestrationTaskPayload {
  id: string;
  name: string;
  task_kind: 'orchestration';
  orchestration: OrchestrationDefinition;
  schedule_type: 'interval' | 'cron';
  interval_seconds?: number;
  cron_expr?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  last_run_at?: string;
  next_run_at?: string;
  last_error?: string;
}

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

export interface TaskRunCard {
  card_id: string;
  run_id?: string;
  kind: string;
  title?: string;
  node_id?: string;
  node_type?: string;
  round?: number;
  iteration?: number;
  branch_id?: string;
  source_session_id?: string;
  started_at?: string;
  status?: string;
  finished_at?: string;
  preview?: string;
  error?: string;
  final_text?: string;
  source_events?: AgentStreamEvent[];
}

export interface WorkflowStartNode {
  inputs?: WorkflowInputVariable[];
}

export interface TaskRunLog {
  task_id: string;
  run_id: string;
  trace_id: string;
  task_kind?: 'agent_message' | 'workflow' | 'orchestration';
  scheduled_at: string;
  started_at?: string;
  finished_at?: string;
  status: 'running' | 'success' | 'incomplete' | 'cancelled' | 'error' | 'skipped' | 'awaiting_human';
  session_id_input?: string;
  session_id_output?: string;
  response_preview?: string;
  node_results?: TaskRunNodeResult[];
  run_cards?: TaskRunCard[];
  error?: string;
}

export interface WorkflowInputVariable {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'object' | 'array';
  required?: boolean;
  default?: unknown;
  description?: string;
}

export interface TaskRunPayload {
  task: TaskPayload;
  run: TaskRunLog;
}

export interface TaskRunStopRequest {
  run_id: string;
}

export interface TaskRunStopResponse {
  status: 'stopped' | 'not_running';
  message: string;
  task_id: string;
  run_id?: string;
  run?: TaskRunLog;
}

export type AgentSendResponse = AgentSendSuccessResponse | AgentSendAwaitingHumanResponse;

export type TaskCreateRequest = AgentMessageTaskCreateRequest | WorkflowTaskCreateRequest | OrchestrationTaskCreateRequest;

export type TaskPayload = AgentMessageTaskPayload | WorkflowTaskPayload | OrchestrationTaskPayload;
