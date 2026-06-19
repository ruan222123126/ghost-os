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
  bridgeUrl: string;
  connectionMode: ConnectionMode;
  pairing?: MobilePairingInfo;
  sessionId: string;
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
  type: string;
  base_url: string;
  models?: string[];
  api_key_set: boolean;
}

export interface ProviderListPayload {
  providers: ProviderConfigPayload[];
  active_provider: string;
}

export interface AgentPayload {
  message: string;
  session_id: string;
  session_ended: boolean;
  thinking?: string;
  mode?: string;
}

export type SessionMessageRole = "system" | "internal" | "user" | "assistant" | "tool";

export interface SessionContentPart {
  type: string;
  text?: string;
}

export interface SessionMessage {
  index: number;
  role: SessionMessageRole;
  text?: string;
  content?: SessionContentPart[];
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
