export type { ApiEnvelope, ApiErrorEnvelope, ApiRequest, ApiSuccessEnvelope } from '@/lib/envelope.generated';

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

export interface ErrorChatMessage {
  id: string;
  kind: 'error';
  content: string;
  error: string;
}

export type ChatMessage = UserChatMessage | AssistantChatMessage | ErrorChatMessage;

export interface SessionMetadata {
  id: string;
  created_at: string;
  updated_at: string;
  message_count: number;
  token_count: number;
}

export interface SessionMessage {
  role: 'system' | 'user' | 'assistant' | 'tool';
  text: string;
  tool_calls?: unknown[];
  tool_results?: unknown[];
}

export interface SessionDetail {
  id: string;
  messages: SessionMessage[];
  created_at: string;
  updated_at: string;
  token_count: number;
}

export interface AgentSendResponse {
  message: string;
  session_id: string;
}

export interface BridgeConfig {
  provider: string;
  base_url: string;
  model: string;
  chat_path: string;
  api_key_set: boolean;
}

export interface ConfigUpdate {
  provider?: string;
  base_url?: string;
  model?: string;
  chat_path?: string;
  api_key?: string;
}
