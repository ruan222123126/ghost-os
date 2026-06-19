import { useState } from "react";
import type { FormEvent, ReactNode } from "react";
import { Check, Pencil, Plus, RefreshCw, Trash2 } from "lucide-react";
import type {
  ConfigPayload,
  ProviderConfigInputPayload,
  ProviderConfigPayload,
  ProviderListPayload,
  ProviderType,
} from "../mobileTypes";
import "./MobileProviderSettings.css";

interface MobileProviderSettingsProps {
  config: ConfigPayload | undefined;
  onActivateProvider: (name: string) => Promise<boolean>;
  onCreateProvider: (provider: ProviderConfigInputPayload) => Promise<boolean>;
  onDeleteProvider: (name: string) => Promise<boolean>;
  onRefreshProviders: () => Promise<boolean>;
  onUpdateProvider: (name: string, provider: ProviderConfigInputPayload) => Promise<boolean>;
  providerList: ProviderListPayload | undefined;
}

interface ProviderEditorState {
  name: string;
  providerType: ProviderType;
  baseURL: string;
  apiKey: string;
  models: string;
  contextWindowTokens: string;
}

type ProviderEditorMode = "create" | "edit";
type ProviderView = "list" | "editor";

const PROVIDER_TYPE_OPTIONS: Array<{
  value: ProviderType;
  label: string;
  defaultBaseURL: string;
}> = [
  { value: "openai", label: "OpenAI", defaultBaseURL: "https://api.openai.com/v1" },
  { value: "codex", label: "Codex", defaultBaseURL: "https://api.openai.com/v1" },
  { value: "anthropic", label: "Anthropic", defaultBaseURL: "https://api.anthropic.com" },
  { value: "custom", label: "OpenAI-Compatible", defaultBaseURL: "" },
];

const EMPTY_EDITOR: ProviderEditorState = {
  name: "",
  providerType: "openai",
  baseURL: "",
  apiKey: "",
  models: "",
  contextWindowTokens: "",
};

export function MobileProviderSettings(props: MobileProviderSettingsProps) {
  const [view, setView] = useState<ProviderView>("list");
  const [editorMode, setEditorMode] = useState<ProviderEditorMode>("create");
  const [editingName, setEditingName] = useState("");
  const [editor, setEditor] = useState<ProviderEditorState>({ ...EMPTY_EDITOR });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const providers = props.providerList?.providers ?? [];
  const activeProvider = props.providerList?.active_provider || props.config?.provider || "";

  function startCreate(): void {
    setEditor({ ...EMPTY_EDITOR });
    setEditorMode("create");
    setEditingName("");
    setError("");
    setView("editor");
  }

  function startEdit(provider: ProviderConfigPayload): void {
    setEditor(editorFromProvider(provider));
    setEditorMode("edit");
    setEditingName(provider.name);
    setError("");
    setView("editor");
  }

  async function submitProvider(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setError("");
    setBusy(true);
    try {
      const input = providerInputFromEditor(editor);
      const ok = editorMode === "edit"
        ? await props.onUpdateProvider(editingName, input)
        : await props.onCreateProvider(input);
      if (ok) {
        setView("list");
      } else {
        setError("供应商保存失败");
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : String(caught));
    } finally {
      setBusy(false);
    }
  }

  if (view === "editor") {
    return (
      <ProviderEditorForm
        busy={busy}
        editor={editor}
        editorMode={editorMode}
        error={error}
        onCancel={() => setView("list")}
        onChange={(patch) => setEditor((current) => ({ ...current, ...patch }))}
        onSelectType={(providerType) => setEditor((current) => nextEditorForType(current, providerType))}
        onSubmit={submitProvider}
      />
    );
  }

  return (
    <ProviderListView
      activeProvider={activeProvider}
      busy={busy}
      error={error}
      model={props.config?.model}
      providers={providers}
      onActivateProvider={(name) => void runProviderAction(setBusy, setError, () => props.onActivateProvider(name))}
      onCreate={startCreate}
      onDeleteProvider={(name) => void deleteProvider(name, setBusy, setError, props.onDeleteProvider)}
      onEdit={startEdit}
      onRefreshProviders={() => void runProviderAction(setBusy, setError, props.onRefreshProviders)}
    />
  );
}

interface ProviderListViewProps {
  activeProvider: string;
  busy: boolean;
  error: string;
  model: string | undefined;
  providers: ProviderConfigPayload[];
  onActivateProvider: (name: string) => void;
  onCreate: () => void;
  onDeleteProvider: (name: string) => void;
  onEdit: (provider: ProviderConfigPayload) => void;
  onRefreshProviders: () => void;
}

function ProviderListView(props: ProviderListViewProps) {
  return (
    <section className="mobile-settings-provider-shell" aria-labelledby="mobile-settings-provider-title">
      <div className="mobile-settings-provider-title-row">
        <div>
          <div className="mobile-settings-connection-title" id="mobile-settings-provider-title">
            供应商
          </div>
          <p className="mobile-settings-provider-summary">{providerSummary(props.activeProvider, props.model)}</p>
        </div>
        <div className="mobile-settings-provider-title-actions">
          <button type="button" aria-label="刷新供应商" disabled={props.busy} onClick={props.onRefreshProviders}>
            <RefreshCw className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          </button>
          <button type="button" aria-label="新增供应商" disabled={props.busy} onClick={props.onCreate}>
            <Plus className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          </button>
        </div>
      </div>

      {props.error ? <p className="mobile-settings-error is-card" role="alert">{props.error}</p> : null}

      {props.providers.length === 0 ? (
        <div className="mobile-settings-empty-card">暂无供应商</div>
      ) : (
        <div className="mobile-settings-provider-list">
          {props.providers.map((provider) => (
            <ProviderCard
              key={provider.name}
              active={stringsEqualIgnoreCase(provider.name, props.activeProvider)}
              busy={props.busy}
              provider={provider}
              onActivate={props.onActivateProvider}
              onDelete={props.onDeleteProvider}
              onEdit={props.onEdit}
            />
          ))}
        </div>
      )}
    </section>
  );
}

interface ProviderCardProps {
  active: boolean;
  busy: boolean;
  provider: ProviderConfigPayload;
  onActivate: (name: string) => void;
  onDelete: (name: string) => void;
  onEdit: (provider: ProviderConfigPayload) => void;
}

function ProviderCard(props: ProviderCardProps) {
  const { active, busy, provider } = props;

  return (
    <article className={`mobile-settings-provider-card ${active ? "is-active" : ""}`}>
      <div className="mobile-settings-provider-card-main">
        <div className="mobile-settings-provider-heading">
          <span>{provider.name}</span>
          {active ? <small>已激活</small> : null}
        </div>
        <p>{labelForProviderType(provider.type)}</p>
        <code>{provider.base_url || defaultBaseURLForProviderType(provider.type) || "未配置 Endpoint"}</code>
        <div className="mobile-settings-provider-meta">
          <span>模型 {provider.models?.length ?? 0}</span>
          <span>{provider.api_key_set ? "API Key 已配置" : "API Key 未配置"}</span>
          {provider.context_window_tokens ? <span>{provider.context_window_tokens} tokens</span> : null}
        </div>
      </div>
      <div className="mobile-settings-provider-actions">
        {active ? null : (
          <button type="button" disabled={busy} onClick={() => props.onActivate(provider.name)}>
            <Check className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
            激活
          </button>
        )}
        <button type="button" aria-label={`编辑 ${provider.name}`} disabled={busy} onClick={() => props.onEdit(provider)}>
          <Pencil className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
        </button>
        <button type="button" aria-label={`删除 ${provider.name}`} disabled={busy} onClick={() => props.onDelete(provider.name)}>
          <Trash2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
        </button>
      </div>
    </article>
  );
}

interface ProviderEditorFormProps {
  busy: boolean;
  editor: ProviderEditorState;
  editorMode: ProviderEditorMode;
  error: string;
  onCancel: () => void;
  onChange: (patch: Partial<ProviderEditorState>) => void;
  onSelectType: (providerType: ProviderType) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>;
}

function ProviderEditorForm(props: ProviderEditorFormProps) {
  const title = props.editorMode === "edit" ? props.editor.name || "编辑供应商" : "新增供应商";
  const submitLabel = props.busy ? "保存中" : props.editorMode === "edit" ? "保存" : "创建";

  return (
    <form className="mobile-settings-provider-form" onSubmit={(event) => void props.onSubmit(event)}>
      <div className="mobile-settings-connection-title">{title}</div>
      {props.error ? <p className="mobile-settings-error is-card" role="alert">{props.error}</p> : null}

      <ProviderField label="供应商名称">
        <input
          value={props.editor.name}
          disabled={props.busy}
          onChange={(event) => props.onChange({ name: event.target.value })}
        />
      </ProviderField>

      <ProviderField label="类型">
        <select
          value={props.editor.providerType}
          disabled={props.busy}
          onChange={(event) => props.onSelectType(event.target.value as ProviderType)}
        >
          {PROVIDER_TYPE_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </ProviderField>

      <ProviderField label="Endpoint">
        <input
          value={props.editor.baseURL}
          disabled={props.busy}
          placeholder={endpointPlaceholder(props.editor.providerType)}
          onChange={(event) => props.onChange({ baseURL: event.target.value })}
        />
      </ProviderField>

      <ProviderField label="API Key">
        <input
          type="password"
          value={props.editor.apiKey}
          disabled={props.busy}
          placeholder={props.editorMode === "edit" ? "留空保留原 Key" : "可留空"}
          onChange={(event) => props.onChange({ apiKey: event.target.value })}
        />
      </ProviderField>

      <ProviderField label="Models">
        <input
          value={props.editor.models}
          disabled={props.busy}
          placeholder="gpt-4o, claude-3-7-sonnet-latest"
          onChange={(event) => props.onChange({ models: event.target.value })}
        />
      </ProviderField>

      <ProviderField label="Context Window Tokens">
        <input
          type="number"
          min="1"
          step="1"
          value={props.editor.contextWindowTokens}
          disabled={props.busy}
          placeholder="可选"
          onChange={(event) => props.onChange({ contextWindowTokens: event.target.value })}
        />
      </ProviderField>

      <div className="mobile-settings-provider-form-actions">
        <button type="button" disabled={props.busy} onClick={props.onCancel}>
          取消
        </button>
        <button type="submit" disabled={props.busy}>
          {submitLabel}
        </button>
      </div>
    </form>
  );
}

function ProviderField(props: { children: ReactNode; label: string }) {
  return (
    <label className="mobile-settings-provider-field">
      <span>{props.label}</span>
      {props.children}
    </label>
  );
}

function editorFromProvider(provider: ProviderConfigPayload): ProviderEditorState {
  return {
    name: provider.name,
    providerType: provider.type,
    baseURL: provider.base_url,
    apiKey: "",
    models: provider.models?.join(", ") ?? "",
    contextWindowTokens: provider.context_window_tokens ? String(provider.context_window_tokens) : "",
  };
}

function nextEditorForType(state: ProviderEditorState, providerType: ProviderType): ProviderEditorState {
  const currentDefault = defaultBaseURLForProviderType(state.providerType);
  const nextDefault = defaultBaseURLForProviderType(providerType);
  const shouldReplaceBaseURL = state.baseURL.trim() === "" || state.baseURL === currentDefault;

  return {
    ...state,
    providerType,
    baseURL: shouldReplaceBaseURL ? nextDefault : state.baseURL,
  };
}

function providerInputFromEditor(editor: ProviderEditorState): ProviderConfigInputPayload {
  const input: ProviderConfigInputPayload = {
    name: editor.name.trim(),
    type: editor.providerType,
    models: parseProviderModels(editor.models),
  };
  if (!input.name) {
    throw new Error("供应商名称不能为空");
  }
  if (editor.baseURL.trim()) {
    input.base_url = editor.baseURL.trim();
  }
  if (editor.apiKey.trim()) {
    input.api_key = editor.apiKey.trim();
  }
  if (editor.contextWindowTokens.trim()) {
    input.context_window_tokens = parsePositiveInteger(editor.contextWindowTokens);
  }
  return input;
}

function parseProviderModels(raw: string): string[] {
  return raw.split(",").map((item) => item.trim()).filter(Boolean);
}

function parsePositiveInteger(raw: string): number {
  const trimmed = raw.trim();
  if (!/^[1-9]\d*$/.test(trimmed)) {
    throw new Error("Context Window Tokens 必须是正整数");
  }
  return Number(trimmed);
}

function defaultBaseURLForProviderType(providerType: ProviderType): string {
  return PROVIDER_TYPE_OPTIONS.find((option) => option.value === providerType)?.defaultBaseURL ?? "";
}

function endpointPlaceholder(providerType: ProviderType): string {
  return providerType === "custom" ? "https://example.com/v1" : defaultBaseURLForProviderType(providerType);
}

function labelForProviderType(providerType: ProviderType): string {
  return PROVIDER_TYPE_OPTIONS.find((option) => option.value === providerType)?.label ?? providerType;
}

function providerSummary(provider: string, model: string | undefined): string {
  if (provider && model) {
    return `${provider} / ${model}`;
  }
  return provider || model || "未激活";
}

function stringsEqualIgnoreCase(left: string, right: string): boolean {
  return left.trim().toLowerCase() === right.trim().toLowerCase();
}

async function runProviderAction(
  setBusy: (busy: boolean) => void,
  setError: (error: string) => void,
  action: () => Promise<boolean>,
): Promise<void> {
  setError("");
  setBusy(true);
  try {
    const ok = await action();
    if (!ok) {
      setError("供应商操作失败");
    }
  } finally {
    setBusy(false);
  }
}

async function deleteProvider(
  name: string,
  setBusy: (busy: boolean) => void,
  setError: (error: string) => void,
  onDeleteProvider: (name: string) => Promise<boolean>,
): Promise<void> {
  if (!window.confirm(`删除供应商 ${name}？`)) {
    return;
  }
  await runProviderAction(setBusy, setError, () => onDeleteProvider(name));
}
