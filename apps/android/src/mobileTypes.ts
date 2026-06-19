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

export interface SessionMetadata {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
  message_count: number;
  token_count: number;
}

export interface StatusMessage {
  tone: "idle" | "loading" | "success" | "error";
  text: string;
}
