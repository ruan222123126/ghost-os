import { useState } from "react";
import type { ReactNode } from "react";
import type { ConfigPayload, HostProfile, StoredSettings } from "../../mobileTypes";
import { DRAWING_PLACEHOLDERS } from "./data";
import { UiIcon } from "./icons";
import type { SidebarHistoryItem, UiIconName } from "./types";

interface MobileSidebarProps {
  open: boolean;
  host: HostProfile | undefined;
  config: ConfigPayload | undefined;
  settings: StoredSettings;
  historyItems: SidebarHistoryItem[];
  activeHistoryId: number | undefined;
  onClose: () => void;
  onNewSession: () => void;
  onConnect: () => Promise<void>;
  onOpenSettings: () => void;
}

export function MobileSidebar(props: MobileSidebarProps) {
  const accountName = props.host?.productName || "Ghost-OS Mobile";
  const isBridgeConnected = Boolean(props.config);
  const bridgeStatusLabel = isBridgeConnected ? "CONNECTED" : "DISCONNECTED";
  const [activeMode, setActiveMode] = useState<"chat" | "drawing">("chat");
  const sortedHistoryItems = [...props.historyItems].sort(compareHistoryItems);

  return (
    <>
      <button
        className={`sidebar-scrim ${props.open ? "is-open" : ""}`}
        type="button"
        aria-label="关闭侧边栏"
        onClick={props.onClose}
      />
      <aside
        className={`mobile-sidebar ${props.open ? "is-open" : ""}`}
        role="dialog"
        aria-hidden={!props.open}
        aria-modal="true"
        inert={props.open ? undefined : true}
      >
        <div className="sidebar-scroll">
          <div className="sidebar-brand">
            <h2>Ghost-OS</h2>
          </div>

          <nav className="sidebar-nav" aria-label="主要操作">
            <SidebarNavButton icon="edit" label="发起新任务" onClick={props.onNewSession} />
            <SidebarNavButton icon="search" label="搜索任务内容" onClick={props.onClose} />
          </nav>

          <div className="sidebar-mode-toggle" role="tablist" aria-label="侧边栏内容切换">
            <button
              className={activeMode === "chat" ? "is-active" : ""}
              type="button"
              role="tab"
              aria-selected={activeMode === "chat"}
              onClick={() => setActiveMode("chat")}
            >
              对话
            </button>
            <button
              className={activeMode === "drawing" ? "is-active" : ""}
              type="button"
              role="tab"
              aria-selected={activeMode === "drawing"}
              onClick={() => setActiveMode("drawing")}
            >
              绘画
            </button>
          </div>

          {activeMode === "chat" ? (
            <SidebarSection title="最近">
              <div className="history-list">
                {sortedHistoryItems.map((item) => {
                  const isActive = props.activeHistoryId === item.id;

                  return (
                    <button
                      key={item.id}
                      className={`history-item ${isActive ? "is-active" : ""} ${item.pinned ? "is-pinned" : ""}`}
                      type="button"
                      title={item.title}
                      onClick={props.onClose}
                    >
                      <span>{item.title}</span>
                      {item.pinned ? (
                        <span className="history-pin-indicator" aria-label="已固定" title="已固定">
                          <UiIcon name="pin" />
                        </span>
                      ) : null}
                    </button>
                  );
                })}
              </div>
            </SidebarSection>
          ) : (
            <SidebarSection title="绘画">
              <div className="drawing-placeholder-list">
                {DRAWING_PLACEHOLDERS.map((item) => (
                  <button key={item.id} className="drawing-placeholder-card" type="button" onClick={props.onClose}>
                    <span>{item.title}</span>
                    <small>{item.desc}</small>
                  </button>
                ))}
              </div>
            </SidebarSection>
          )}
        </div>

        <footer className="sidebar-footer">
          <div className="sidebar-account">
            <div className="sidebar-account-copy">
              <span>{accountName}</span>
              <strong className={isBridgeConnected ? "is-connected" : undefined}>{bridgeStatusLabel}</strong>
            </div>
          </div>
          <button
            className="sidebar-settings-button"
            type="button"
            aria-label="打开设置"
            title={props.settings.bridgeUrl}
            onClick={props.onOpenSettings}
          >
            <UiIcon name="settings" />
          </button>
        </footer>
      </aside>
    </>
  );
}

function compareHistoryItems(a: SidebarHistoryItem, b: SidebarHistoryItem): number {
  if (a.pinned !== b.pinned) {
    return a.pinned ? -1 : 1;
  }
  return a.id - b.id;
}

function SidebarSection(props: { title: string; children: ReactNode }) {
  return (
    <section className="sidebar-section">
      <h3>{props.title}</h3>
      {props.children}
    </section>
  );
}

function SidebarNavButton(props: { icon: UiIconName; label: string; onClick: () => void }) {
  return (
    <button className="sidebar-nav-button" type="button" onClick={props.onClick}>
      <UiIcon name={props.icon} />
      <span>{props.label}</span>
    </button>
  );
}
