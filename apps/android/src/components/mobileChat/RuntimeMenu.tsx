import type { ConfigPayload, ProviderConfigPayload, ProviderListPayload, StatusMessage } from "../../mobileTypes";
import { UiIcon } from "./icons";
import "./RuntimeMenu.css";

interface RuntimeMenuProps {
  config: ConfigPayload | undefined;
  providerList: ProviderListPayload | undefined;
  status: StatusMessage;
  bridgeUrl: string;
  onClose: () => void;
  onSwitchModel: (model: string) => Promise<boolean>;
  onOpenSettings: () => void;
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
      desc: props.status.text,
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

export function RuntimeMenu(props: RuntimeMenuProps) {
  const options = modelOptions(props);

  async function selectModel(option: ModelOption): Promise<void> {
    if (option.selected) {
      props.onClose();
      return;
    }
    if (option.disabled) {
      return;
    }
    const didSwitch = await props.onSwitchModel(option.model);
    if (didSwitch) {
      props.onClose();
    }
  }

  return (
    <>
      <button className="runtime-menu-scrim" type="button" aria-label="关闭 Runtime 菜单" onClick={props.onClose} />
      <div className="runtime-menu" role="dialog" aria-label="模型选择">
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
        <div className="runtime-menu-separator" />
        <button
          className="runtime-menu-action"
          type="button"
          title={props.bridgeUrl}
          onClick={() => {
            props.onClose();
            props.onOpenSettings();
          }}
        >
          <span>连接设置</span>
          <UiIcon name="chevron-down" />
        </button>
      </div>
    </>
  );
}
