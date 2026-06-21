export interface HostProfile {
  productName: string;
  version: string;
  target: string;
  mobile: boolean;
}

export type ConnectionMode = "webrtc" | "http";

export interface MobilePairingInfo {
  deviceId: string;
  pcId: string;
  signalingUrl: string;
  signalingToken: string;
  iceServers: RTCIceServer[];
}

export interface StoredConnectionSnapshot {
  apiToken?: string;
  bridgeUrl: string;
  connectionMode: ConnectionMode;
  pairing?: MobilePairingInfo;
}

export interface StoredSettings {
  apiToken?: string;
  autoConnectEnabled: boolean;
  bridgeUrl: string;
  connectionMode: ConnectionMode;
  lastSuccessfulConnection?: StoredConnectionSnapshot;
  pairing?: MobilePairingInfo;
  persistComputerSessionsEnabled: boolean;
}

export interface ConfigPayload {
  provider?: string;
  provider_type?: string;
  model?: string;
  project_root?: string;
  api_key_set?: boolean;
  model_selection_enabled?: boolean;
  task_execution_timeout_ms?: number;
  relay_default_stop_policy?: TaskRelayStopPolicy;
  relay_default_max_rounds?: number;
  relay_default_execution_timeout_ms?: number;
}

export interface ProviderConfigPayload {
  name: string;
  type: ProviderType;
  base_url: string;
  models?: string[];
  context_window_tokens?: number;
  response_reserve_tokens?: number;
  model_context_window_tokens?: Record<string, number>;
  model_response_reserve_tokens?: Record<string, number>;
  api_key_set: boolean;
}

export type ProviderType = "openai" | "anthropic" | "custom" | "codex";

export interface ProviderConfigInputPayload {
  name: string;
  type: ProviderType;
  base_url?: string;
  api_key?: string;
  models?: string[];
  context_window_tokens?: number;
  response_reserve_tokens?: number;
  model_context_window_tokens?: Record<string, number>;
  model_response_reserve_tokens?: Record<string, number>;
}

export interface ProviderListPayload {
  providers: ProviderConfigPayload[];
  active_provider: string;
}

export type SkillSource = "repo" | "user";

export interface SkillPayload {
  id: string;
  name: string;
  description: string;
  path: string;
  source: SkillSource;
  enabled: boolean;
}

export type TaskScheduleType = "interval" | "cron";
export type TaskAgentMode = "single" | "relay";
export type TaskRelayStopPolicy = "ai_decides" | "max_rounds";

export interface TaskRelayConfigPayload {
  stop_policy: TaskRelayStopPolicy;
  max_rounds: number;
  execution_timeout_ms?: number;
}

export interface TaskRuntimeOverridesPayload {
  provider_name?: string;
  model?: string;
  system_prompt?: string;
  preset_id?: string;
  tool_allowlist?: string[];
  tool_allowlist_only?: boolean;
  max_turns?: number;
}

export interface AgentMessageTaskPayload {
  id: string;
  message: string;
  session_id?: string;
  runtime_overrides?: TaskRuntimeOverridesPayload;
  agent_mode?: TaskAgentMode;
  relay?: TaskRelayConfigPayload;
  task_kind: "agent_message";
  schedule_type: TaskScheduleType;
  interval_seconds?: number;
  cron_expr?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  last_run_at?: string;
  next_run_at?: string;
  last_error?: string;
}

export interface WorkflowNode {
  id: string;
  type: string;
  [key: string]: unknown;
}

export interface WorkflowDefinition {
  nodes: WorkflowNode[];
  edges?: Array<Record<string, unknown>>;
}

export interface WorkflowTaskPayload {
  id: string;
  task_kind: "workflow";
  workflow: WorkflowDefinition;
  schedule_type: TaskScheduleType;
  interval_seconds?: number;
  cron_expr?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  last_run_at?: string;
  next_run_at?: string;
  last_error?: string;
}

export type TaskPayload = AgentMessageTaskPayload | WorkflowTaskPayload;

export type UnknownTaskPayload = {
  id?: string;
  task_kind?: string;
  agent_mode?: string;
  [key: string]: unknown;
};

export interface AgentPayload {
  message: string;
  session_id: string;
  session_ended: boolean;
  thinking?: string;
  mode?: string;
  tools?: MobileToolCard[];
}

export type SessionMessageRole = "system" | "internal" | "user" | "assistant" | "tool";

export interface SessionContentPart {
  type: string;
  text?: string;
}

export interface SessionToolCall {
  id: string;
  name: string;
  arguments: Record<string, unknown>;
}

export type SessionToolResultStatus = "success" | "error";

export interface SessionToolResult {
  status: SessionToolResultStatus;
  tool: string;
  trace_id?: string;
  output?: string;
  error?: string;
}

export interface SessionMessage {
  index: number;
  role: SessionMessageRole;
  text?: string;
  content?: SessionContentPart[];
  tool_calls?: SessionToolCall[];
  tool_result?: SessionToolResult | null;
  tool_call_id?: string;
  in_progress?: boolean;
  thinking?: string;
}

export interface SessionMessagePage {
  limit: number;
  before?: number | null;
  start_index?: number | null;
  end_index?: number | null;
  has_more_before: boolean;
  next_before?: number | null;
}

export interface SessionMetadata {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
  message_count: number;
  token_count: number;
}

export interface SessionDetail extends SessionMetadata {
  messages: SessionMessage[];
  page: SessionMessagePage;
}

export interface MobileConversationMessage {
  id: string;
  role: "user" | "assistant";
  text: string;
  thinking?: string;
  sessionId?: string;
  tools?: MobileToolCard[];
}

export type MobileToolCardStatus = "pending" | "running" | "success" | "error";

export interface MobileToolCard {
  id: string;
  toolName?: string;
  toolCallId?: string;
  status: MobileToolCardStatus;
  input?: string;
  output?: string;
  error?: string;
  traceId?: string;
}

export interface StoredMobileConversation {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
  messages: MobileConversationMessage[];
  source_message_count?: number;
}

export interface StatusMessage {
  tone: "idle" | "loading" | "success" | "error";
  text: string;
}

export type MobileSessionRunStatus = "idle" | "running" | "success" | "error";

export interface MobileSessionRunState {
  requestId?: string;
  sessionEnded: boolean;
  status: MobileSessionRunStatus;
  statusText: string;
  stopPending?: boolean;
  traceId?: string;
}

export interface MobileSessionView {
  bridgeOwned: boolean;
  id: string;
  messages: MobileConversationMessage[];
  reply?: AgentPayload;
  run: MobileSessionRunState;
  title: string;
  unread: boolean;
  updatedAt: string;
}
