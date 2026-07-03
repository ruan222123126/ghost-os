import type {
  AskHumanOption as SharedAskHumanOption,
  SessionContentPart as SharedSessionContentPart,
  SessionDetail as SharedSessionDetail,
  SessionMessage as SharedSessionMessage,
  SessionMessagePage as SharedSessionMessagePage,
  SessionMetadata as SharedSessionMetadata,
  SessionRuntimeSelection as SharedSessionRuntimeSelection,
  SessionToolCall as SharedSessionToolCall,
  SessionToolResult as SharedSessionToolResult,
  SessionTurnDraft as SharedSessionTurnDraft,
  SessionTurnDraftPendingQuestion as SharedSessionTurnDraftPendingQuestion,
  SessionTurnDraftSegment as SharedSessionTurnDraftSegment,
  SessionTurnDraftTool as SharedSessionTurnDraftTool,
  TaskRuntimeOverrides as SharedTaskRuntimeOverrides,
} from "./lib/envelope.generated";

export interface HostProfile {
  productName: string;
  version: string;
  target: string;
  mobile: boolean;
}

export type ConnectionMode = "webrtc" | "http";
export type AgentRuntimeType = "ghost" | "codex";
export type AgentModeSelection = "normal" | "plan" | null;
export type AgentRequestMode = "plan";
export type ExternalCodexPermissionMode = "read-only" | "default" | "safe-yolo" | "yolo";
export type ExternalAgentApprovalDecision = "approved" | "approved_for_session" | "denied" | "abort";

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
  localProviderId?: string;
  localModel?: string;
  pairing?: MobilePairingInfo;
  persistComputerSessionsEnabled: boolean;
  remoteExecutionEnabled: boolean;
}

export interface ConfigPayload {
  provider?: string;
  provider_type?: string;
  model?: string;
  project_root?: string;
  external_codex_permission_mode?: ExternalCodexPermissionMode;
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
  provider_id: string;
  updated_at: string;
  deleted_at?: string;
  models?: string[];
  context_window_tokens?: number;
  response_reserve_tokens?: number;
  model_context_window_tokens?: Record<string, number>;
  model_response_reserve_tokens?: Record<string, number>;
  api_key_set: boolean;
  sync_state?: "local" | "synced";
}

export type ProviderType = "openai" | "anthropic" | "custom" | "codex";

export interface ProviderConfigInputPayload {
  name: string;
  type: ProviderType;
  provider_id?: string;
  updated_at?: string;
  deleted_at?: string;
  base_url?: string;
  api_key?: string;
  models?: string[];
  context_window_tokens?: number;
  response_reserve_tokens?: number;
  model_context_window_tokens?: Record<string, number>;
  model_response_reserve_tokens?: Record<string, number>;
}

export interface ProviderExportRequestPayload extends Record<string, unknown> {
  name?: string;
  provider_id?: string;
}

export interface ProviderSyncRecordPayload {
  provider_id: string;
  updated_at: string;
  deleted_at?: string;
  name?: string;
  type?: ProviderType;
  base_url?: string;
  models?: string[];
  context_window_tokens?: number;
  response_reserve_tokens?: number;
  model_context_window_tokens?: Record<string, number>;
  model_response_reserve_tokens?: Record<string, number>;
  api_key_set?: boolean;
}

export interface ProviderListPayload {
  providers: ProviderConfigPayload[];
  active_provider: string;
  active_provider_id?: string;
  provider_sync_records?: ProviderSyncRecordPayload[];
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

export type TaskRuntimeOverridesContract = SharedTaskRuntimeOverrides;

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

export interface MobileAssistantTextPart {
  id: string;
  kind: "text";
  text: string;
}

export interface MobileAssistantToolPart {
  id: string;
  kind: "tool";
  tool: MobileToolCard;
}

export type MobileAssistantPart = MobileAssistantTextPart | MobileAssistantToolPart;

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

export type OrchestrationNodeType = "group" | "agent";
export type OrchestrationEdgeKind = "control" | "member";
export type OrchestrationSpeakingMode = "sequential" | "parallel" | "owner";

export interface OrchestrationGroupNodePayload {
  title: string;
  shared_context?: string;
  speaking_mode: OrchestrationSpeakingMode;
  owner_agent_id?: string;
  max_rounds: number;
}

export interface OrchestrationAgentNodePayload {
  title: string;
  message: string;
  runtime_overrides?: TaskRuntimeOverridesPayload;
}

export interface OrchestrationNodePayload {
  id: string;
  type: OrchestrationNodeType;
  group?: OrchestrationGroupNodePayload;
  agent?: OrchestrationAgentNodePayload;
}

export interface OrchestrationEdgePayload {
  from_node_id: string;
  to_node_id: string;
  kind: OrchestrationEdgeKind;
}

export interface OrchestrationDefinitionPayload {
  nodes: OrchestrationNodePayload[];
  edges: OrchestrationEdgePayload[];
}

export interface OrchestrationTaskPayload {
  id: string;
  name: string;
  task_kind: "orchestration";
  orchestration: OrchestrationDefinitionPayload;
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
  parts?: MobileAssistantPart[];
  tools?: MobileToolCard[];
}

export type SessionMessageRole = SharedSessionMessage["role"];
export type SessionContentPart = SharedSessionContentPart;
export type SessionToolCall = SharedSessionToolCall;
export type SessionToolResultStatus = SharedSessionToolResult["status"];
export type SessionToolResult = SharedSessionToolResult;
export type SessionMessage = SharedSessionMessage;
export type SessionMessagePage = SharedSessionMessagePage;

export interface SessionGetOptions {
  before?: number;
  limit?: number;
}

export type SessionMetadata = SharedSessionMetadata;
export type AskHumanOption = SharedAskHumanOption;
export type SessionRuntimeSelection = SharedSessionRuntimeSelection;

export type SessionTurnDraftStatus = "streaming" | "awaiting_human" | "error";

export type SessionTurnDraftSegment = SharedSessionTurnDraftSegment;
export type SessionTurnDraftTool = SharedSessionTurnDraftTool;
export type SessionTurnDraftPendingQuestion = SharedSessionTurnDraftPendingQuestion;
export type SessionTurnDraft = SharedSessionTurnDraft;
export type SessionDetail = SharedSessionDetail;

export interface ChatSelectedSkill {
  id: string;
  name: string;
}

export interface MobileConversationMessage {
  id: string;
  role: "user" | "assistant";
  parts?: MobileAssistantPart[];
  selectedSkill?: ChatSelectedSkill;
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
  approvalDecision?: ExternalAgentApprovalDecision;
  approvalId?: string;
  approvalInFlight?: boolean;
  approvalKind?: string;
  approvalPayload?: Record<string, unknown>;
  approvalPrompt?: string;
}

export interface StoredMobileConversation {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
  messages: MobileConversationMessage[];
  source_message_count?: number;
  synced_message_count?: number;
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
  hasOlderHistory?: boolean;
  id: string;
  loadingOlderHistory?: boolean;
  messages: MobileConversationMessage[];
  nextHistoryBefore?: number | null;
  reply?: AgentPayload;
  run: MobileSessionRunState;
  title: string;
  unread: boolean;
  updatedAt: string;
}
