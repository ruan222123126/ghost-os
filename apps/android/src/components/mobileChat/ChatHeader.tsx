import type { ConfigPayload, ProviderListPayload, StatusMessage } from "../../mobileTypes";
import { IconButton, UiIcon } from "./icons";
import { RuntimeMenu } from "./RuntimeMenu";

interface ChatHeaderProps {
  runtimeLabel: string;
  config: ConfigPayload | undefined;
  providerList: ProviderListPayload | undefined;
  status: StatusMessage;
  bridgeUrl: string;
  hasConversation: boolean;
  runtimeMenuOpen: boolean;
  onOpenSidebar: () => void;
  onToggleRuntimeMenu: () => void;
  onCloseRuntimeMenu: () => void;
  onSwitchModel: (model: string) => Promise<boolean>;
  onOpenSettings: () => void;
  onOpenMoreMenu: () => void;
  onNewSession: () => void;
}

type HeaderRuntimeSelectorProps = Omit<
  ChatHeaderProps,
  "hasConversation" | "onOpenSidebar" | "onOpenMoreMenu" | "onNewSession"
>;

function HeaderRuntimeSelector(props: HeaderRuntimeSelectorProps) {
  return (
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
          providerList={props.providerList}
          status={props.status}
          bridgeUrl={props.bridgeUrl}
          onClose={props.onCloseRuntimeMenu}
          onSwitchModel={props.onSwitchModel}
          onOpenSettings={props.onOpenSettings}
        />
      ) : null}
    </div>
  );
}

export function ChatHeader(props: ChatHeaderProps) {
  if (props.hasConversation) {
    return (
      <header className="chat-header chat-session-header">
        <div className="header-left">
          <IconButton label="打开侧边栏" icon="menu" onClick={props.onOpenSidebar} />
          <HeaderRuntimeSelector {...props} />
        </div>
        <div className="header-right">
          <IconButton label="新会话" icon="edit" onClick={props.onNewSession} />
          <IconButton label="更多操作" icon="more" onClick={props.onOpenMoreMenu} />
        </div>
      </header>
    );
  }

  return (
    <header className="chat-header chat-home-header">
      <div className="header-left">
        <IconButton label="打开侧边栏" icon="menu" onClick={props.onOpenSidebar} />
        <HeaderRuntimeSelector {...props} />
      </div>
      <div className="header-right">
        <IconButton label="新会话" icon="edit" onClick={props.onNewSession} />
      </div>
    </header>
  );
}
