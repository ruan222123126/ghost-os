export interface CodexModelCatalogLike {
  models: string[];
  default_model: string;
}

export const EMPTY_CODEX_MODEL_CATALOG: CodexModelCatalogLike = {
  models: [],
  default_model: "",
};

export function normalizeCodexModel(
  model: string | undefined,
  catalog: CodexModelCatalogLike = EMPTY_CODEX_MODEL_CATALOG,
): string {
  const trimmed = model?.trim() ?? "";
  if (trimmed && (catalog.models.length === 0 || catalog.models.includes(trimmed))) {
    return trimmed;
  }
  return catalog.default_model.trim() || catalog.models[0]?.trim() || "";
}
