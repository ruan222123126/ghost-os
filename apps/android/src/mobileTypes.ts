export interface HostProfile {
  productName: string;
  version: string;
  target: string;
  mobile: boolean;
}

export interface StoredSettings {
  bridgeUrl: string;
  sessionId: string;
}

export interface ConfigPayload {
  provider?: string;
  provider_type?: string;
  model?: string;
  project_root?: string;
  api_key_set?: boolean;
}

export interface AgentPayload {
  message: string;
  session_id: string;
  session_ended: boolean;
  mode?: string;
}

export interface StatusMessage {
  tone: "idle" | "loading" | "success" | "error";
  text: string;
}
