import type {
  AgentRuntimeType,
  ConfigPayload,
  CodexModelCatalogPayload,
  ExternalCodexPermissionMode,
  ProviderConfigPayload,
  ProviderListPayload,
  StatusMessage,
} from "../../mobileTypes";
import { EMPTY_CODEX_MODEL_CATALOG, normalizeCodexModel } from "../../lib/codexModels";
import { UiIcon } from "./icons";
import "./RuntimeMenu.css";

interface RuntimeMenuProps {
  agentRuntime: AgentRuntimeType;
  codexModel?: string;
  codexModelCatalog?: CodexModelCatalogPayload;
  codexModelCatalogError?: string;
  codexPermissionMode?: ExternalCodexPermissionMode;
  config: ConfigPayload | undefined;
  providerList: ProviderListPayload | undefined;
  status: StatusMessage;
  open: boolean;
  onClose: () => void;
  onSwitchAgentRuntime: (runtime: AgentRuntimeType) => void;
  onSwitchCodexModel?: (model: string) => void;
  onSwitchModel: (model: string) => Promise<boolean>;
}

interface ModelOption {
  id: string;
  model: string;
  desc: string;
  selected: boolean;
  disabled: boolean;
}

function equalName(left: string | undefined, right: string | undefined): boolean {
  return (left ?? "").trim().toLowerCase() === (right ?? "").trim().toLowerCase();
}

function findActiveProvider(
  config: ConfigPayload | undefined,
  providerList: ProviderListPayload | undefined,
): ProviderConfigPayload | undefined {
  const activeName = config?.provider || providerList?.active_provider;
  return providerList?.providers.find((provider) => equalName(provider.name, activeName));
}

function uniqueModels(currentModel: string | undefined, provider: ProviderConfigPayload | undefined): string[] {
  const models = [currentModel, ...(provider?.models ?? [])]
    .map((model) => model?.trim() ?? "")
    .filter((model) => model.length > 0);
  return [...new Set(models)];
}

function modelOptions(props: RuntimeMenuProps): ModelOption[] {
  const provider = findActiveProvider(props.config, props.providerList);
  const providerName = provider?.name || props.config?.provider || "Bridge";
  const models = uniqueModels(props.config?.model, provider);
  const canSwitch = Boolean(props.config && props.providerList && props.config.model_selection_enabled !== false);

  if (models.length === 0) {
    return [{
      id: "not-connected",
      model: "未连接",
      desc: props.codexModelCatalogError || props.status.text,
      selected: true,
      disabled: true,
    }];
  }

  return models.map((model) => {
    const selected = model === props.config?.model;
    return {
      id: `${providerName}:${model}`,
      model,
      desc: selected ? "当前主模型" : providerName,
      selected,
      disabled: !canSwitch || props.status.tone === "loading",
    };
  });
}

function codexModelOptions(props: RuntimeMenuProps): ModelOption[] {
  const catalog = props.codexModelCatalog ?? EMPTY_CODEX_MODEL_CATALOG;
  const activeModel = normalizeCodexModel(props.codexModel, catalog);
  if (catalog.models.length === 0) {
    return [{
      id: "codex-models-unavailable",
      model: "暂无可用模型",
      desc: props.status.text,
      selected: true,
      disabled: true,
    }];
  }
  return catalog.models.map((model) => {
    const selected = model === activeModel;
    return {
      id: `codex:${model}`,
      model,
      desc: selected ? "当前 Codex 模型" : "Codex",
      selected,
      disabled: props.status.tone === "loading",
    };
  });
}

export function RuntimeMenu(props: RuntimeMenuProps) {
  const options = props.agentRuntime === "codex" ? codexModelOptions(props) : modelOptions(props);

  function selectAgentRuntime(runtime: AgentRuntimeType): void {
    props.onSwitchAgentRuntime(runtime);
    props.onClose();
  }

  async function selectModel(option: ModelOption): Promise<void> {
    if (option.selected) {
      props.onClose();
      return;
    }
    if (option.disabled) {
      return;
    }
    if (props.agentRuntime === "codex") {
      props.onSwitchCodexModel?.(option.model);
      props.onClose();
      return;
    }
    const didSwitch = await props.onSwitchModel(option.model);
    if (didSwitch) {
      props.onClose();
    }
  }

  return (
    <>
      <button
        className={`runtime-menu-scrim ${props.open ? "is-open" : "is-closing"}`}
        type="button"
        aria-label="关闭 Runtime 菜单"
        onClick={props.onClose}
      />
      <div className={`runtime-menu ${props.open ? "is-open" : "is-closing"}`} role="dialog" aria-label="模型选择">
        <div className="runtime-agent-options" role="group" aria-label="Agent 类型">
          <button
            className={props.agentRuntime === "ghost" ? "is-selected" : ""}
            type="button"
            aria-pressed={props.agentRuntime === "ghost"}
            onClick={() => selectAgentRuntime("ghost")}
          >
            <span className="runtime-check">{props.agentRuntime === "ghost" ? <UiIcon name="check" /> : null}</span>
            <span className="runtime-option-copy">
              <strong>Ghost</strong>
              <span>Bridge Runtime</span>
            </span>
          </button>
          <button
            className={props.agentRuntime === "codex" ? "is-selected" : ""}
            type="button"
            aria-pressed={props.agentRuntime === "codex"}
            onClick={() => selectAgentRuntime("codex")}
          >
            <span className="runtime-check">{props.agentRuntime === "codex" ? <UiIcon name="check" /> : null}</span>
            <span className="runtime-option-copy">
              <strong>Codex</strong>
              <span>
                {normalizeCodexModel(
                  props.codexModel,
                  props.codexModelCatalog ?? EMPTY_CODEX_MODEL_CATALOG,
                ) || "自动"} / {props.codexPermissionMode ?? "default"}
              </span>
            </span>
          </button>
        </div>
        <div className="runtime-menu-options">
          {options.map((option) => (
            <button
              key={option.id}
              className={`runtime-menu-option ${option.selected ? "is-selected" : ""}`}
              type="button"
              disabled={option.disabled && !option.selected}
              title={option.model}
              onClick={() => void selectModel(option)}
            >
              <span className="runtime-check">{option.selected ? <UiIcon name="check" /> : null}</span>
              <span className="runtime-option-copy">
                <strong>{option.model}</strong>
                <span>{option.desc}</span>
              </span>
            </button>
          ))}
        </div>
      </div>
    </>
  );
}
