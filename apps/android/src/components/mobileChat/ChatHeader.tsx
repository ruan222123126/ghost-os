import { useEffect, useState } from "react";
import type { ConfigPayload, ProviderListPayload, StatusMessage } from "../../mobileTypes";
import { IconButton, UiIcon } from "./icons";
import { RuntimeMenu } from "./RuntimeMenu";

const RUNTIME_MENU_ANIMATION_MS = 180;

interface ChatHeaderProps {
  runtimeLabel: string;
  config: ConfigPayload | undefined;
  providerList: ProviderListPayload | undefined;
  status: StatusMessage;
  hasConversation: boolean;
  runtimeMenuOpen: boolean;
  onOpenSidebar: () => void;
  onToggleRuntimeMenu: () => void;
  onCloseRuntimeMenu: () => void;
  onSwitchModel: (model: string) => Promise<boolean>;
  onOpenMoreMenu: () => void;
  onNewSession: () => void;
}

type HeaderRuntimeSelectorProps = Omit<
  ChatHeaderProps,
  "hasConversation" | "onOpenSidebar" | "onOpenMoreMenu" | "onNewSession"
>;

function HeaderRuntimeSelector(props: HeaderRuntimeSelectorProps) {
  const [renderMenu, setRenderMenu] = useState(props.runtimeMenuOpen);

  useEffect(() => {
    if (props.runtimeMenuOpen) {
      setRenderMenu(true);
      return undefined;
    }
    if (!renderMenu) {
      return undefined;
    }

    const timeout = window.setTimeout(() => setRenderMenu(false), RUNTIME_MENU_ANIMATION_MS);
    return () => window.clearTimeout(timeout);
  }, [props.runtimeMenuOpen, renderMenu]);

  const shouldRenderMenu = props.runtimeMenuOpen || renderMenu;

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

      {shouldRenderMenu ? (
        <RuntimeMenu
          config={props.config}
          providerList={props.providerList}
          status={props.status}
          open={props.runtimeMenuOpen}
          onClose={props.onCloseRuntimeMenu}
          onSwitchModel={props.onSwitchModel}
        />
      ) : null}
    </div>
  );
}

function hasRuntimeModel(config: ConfigPayload | undefined): boolean {
  return Boolean(config?.model?.trim());
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

  const showHomeRuntimeSelector = hasRuntimeModel(props.config);

  return (
    <header className="chat-header chat-home-header">
      <div className="header-left">
        <IconButton label="打开侧边栏" icon="menu" onClick={props.onOpenSidebar} />
        {showHomeRuntimeSelector ? <HeaderRuntimeSelector {...props} /> : null}
      </div>
      <div className="header-right">
        <IconButton label="新会话" icon="edit" onClick={props.onNewSession} />
      </div>
    </header>
  );
}
