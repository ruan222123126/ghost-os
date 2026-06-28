export const CODEX_MODEL_IDS = ["gpt-5.5", "gpt-5.4"] as const;
export const DEFAULT_CODEX_MODEL = CODEX_MODEL_IDS[0];

export function normalizeCodexModel(model: string | undefined): string {
  const trimmed = model?.trim() ?? "";
  return CODEX_MODEL_IDS.includes(trimmed as typeof CODEX_MODEL_IDS[number])
    ? trimmed
    : DEFAULT_CODEX_MODEL;
}
