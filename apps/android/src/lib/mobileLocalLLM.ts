import { invoke } from "@tauri-apps/api/core";
import type { ProviderConfigPayload } from "../mobileTypes";

interface LocalLLMMessage {
  role: "user" | "assistant";
  text: string;
}

interface LocalLLMResponse {
  message: string;
  provider_id: string;
  model: string;
}

export async function sendLocalLLMMessage(input: {
  history: LocalLLMMessage[];
  model: string;
  provider: ProviderConfigPayload;
  sessionId: string;
  traceId: string;
}): Promise<LocalLLMResponse> {
  return invoke<LocalLLMResponse>("mobile_local_llm_send", {
    request: {
      history: input.history,
      model: input.model,
      providerId: input.provider.provider_id,
      sessionId: input.sessionId,
      traceId: input.traceId,
    },
  });
}
