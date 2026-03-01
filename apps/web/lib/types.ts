// Shared web-side TypeScript contracts aligned with bridge API payloads.

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

export interface PendingQuestionMessage {
  id: string;
  kind: 'pending_question';
  content: string;
  questionId: string;
  sessionId: string;
}

export type ChatMessage = UserChatMessage | AssistantChatMessage | ErrorChatMessage | PendingQuestionMessage;

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

export interface AgentSendSuccessResponse {
  message: string;
  session_id: string;
  status?: 'success';
}

export interface AgentSendAwaitingHumanResponse {
  session_id: string;
  status: 'awaiting_human';
  question_id: string;
  prompt: string;
  message?: string;
}

export type AgentSendResponse = AgentSendSuccessResponse | AgentSendAwaitingHumanResponse;

export interface HumanResponseRequest {
  session_id: string;
  question_id: string;
  answer: string;
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
