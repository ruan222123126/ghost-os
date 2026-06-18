import type { ConfigPayload, StatusMessage } from "../../mobileTypes";
import { IconButton, UiIcon } from "./icons";
import { RuntimeMenu } from "./RuntimeMenu";

interface ChatHeaderProps {
  runtimeLabel: string;
  config: ConfigPayload | undefined;
  status: StatusMessage;
  bridgeUrl: string;
  hasConversation: boolean;
  runtimeMenuOpen: boolean;
  onOpenSidebar: () => void;
  onToggleRuntimeMenu: () => void;
  onCloseRuntimeMenu: () => void;
  onOpenSettings: () => void;
  onOpenMoreMenu: () => void;
  onNewSession: () => void;
}

export function ChatHeader(props: ChatHeaderProps) {
  if (props.hasConversation) {
    return (
      <header className="chat-header chat-session-header">
        <div className="header-left">
          <IconButton label="打开侧边栏" icon="menu" onClick={props.onOpenSidebar} />
          <div className="runtime-selector-wrap">
            <button
              className={`runtime-selector ${props.runtimeMenuOpen ? "is-open" : ""}`}
              type="button"
              title={props.runtimeLabel}
              aria-expanded={props.runtimeMenuOpen}
              onClick={props.onToggleRuntimeMenu}
            >
              <span>{props.runtimeLabel}</span>
              <UiIcon name="chevron-down" />
            </button>

            {props.runtimeMenuOpen ? (
              <RuntimeMenu
                config={props.config}
                status={props.status}
                bridgeUrl={props.bridgeUrl}
                runtimeLabel={props.runtimeLabel}
                onClose={props.onCloseRuntimeMenu}
                onOpenSettings={props.onOpenSettings}
              />
            ) : null}
          </div>
        </div>
        <div className="header-right">
          <IconButton label="新会话" icon="edit" onClick={props.onNewSession} />
          <IconButton label="更多操作" icon="more" onClick={props.onOpenMoreMenu} />
        </div>
      </header>
    );
  }

  return (
    <header className="chat-header">
      <IconButton label="打开侧边栏" icon="menu" onClick={props.onOpenSidebar} />
      <h1 className="chat-header-title">Ghost-OS</h1>
      <span className="chat-header-spacer" aria-hidden="true" />
    </header>
  );
}
