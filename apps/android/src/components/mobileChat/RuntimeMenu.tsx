import type { ConfigPayload, StatusMessage } from "../../mobileTypes";
import { UiIcon } from "./icons";
import { runtimeOptions } from "./runtimeOptions";
import "./RuntimeMenu.css";

interface RuntimeMenuProps {
  config: ConfigPayload | undefined;
  status: StatusMessage;
  bridgeUrl: string;
  runtimeLabel: string;
  onClose: () => void;
  onOpenSettings: () => void;
}

export function RuntimeMenu(props: RuntimeMenuProps) {
  const options = runtimeOptions(props);

  return (
    <>
      <button className="runtime-menu-scrim" type="button" aria-label="关闭 Runtime 菜单" onClick={props.onClose} />
      <div className="runtime-menu" role="dialog" aria-label="Runtime 选择">
        <div className="runtime-menu-options">
          {options.map((option, index) => (
            <button
              key={option.id}
              className={`runtime-menu-option ${index === 0 ? "is-selected" : ""}`}
              type="button"
              onClick={props.onClose}
            >
              <span className="runtime-check">{index === 0 ? <UiIcon name="check" /> : null}</span>
              <span className="runtime-option-copy">
                <strong>{option.name}</strong>
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
          <span>执行等级</span>
          <UiIcon name="chevron-down" />
        </button>
      </div>
    </>
  );
}
