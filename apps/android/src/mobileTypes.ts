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

export interface StoredSettings {
  apiToken?: string;
  bridgeUrl: string;
  connectionMode: ConnectionMode;
  pairing?: MobilePairingInfo;
}

export interface ConfigPayload {
  provider?: string;
  provider_type?: string;
  model?: string;
  project_root?: string;
  api_key_set?: boolean;
  model_selection_enabled?: boolean;
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
