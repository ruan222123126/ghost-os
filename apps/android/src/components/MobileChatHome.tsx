import { useEffect, useRef, useState } from "react";
import type { FormEvent, ReactNode, RefObject } from "react";
import {
  ArrowDown,
  ArrowUp,
  Check,
  ChevronDown,
  Diamond,
  HelpCircle,
  LayoutGrid,
  Menu,
  MessageSquareWarning,
  Mic,
  MoreVertical,
  Pencil,
  Pin,
  Plus,
  Search,
  Settings,
  Share2,
  SquarePen,
  Trash2,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { AgentPayload, ConfigPayload, HostProfile, StatusMessage, StoredSettings } from "../mobileTypes";

type UiIconName =
  | "arrow-down"
  | "arrow-up"
  | "check"
  | "chevron-down"
  | "diamond"
  | "edit"
  | "grid"
  | "help"
  | "menu"
  | "mic"
  | "more"
  | "pencil"
  | "pin"
  | "plus"
  | "search"
  | "settings"
  | "share"
  | "trash"
  | "warning";

const ICONS: Record<UiIconName, LucideIcon> = {
  "arrow-down": ArrowDown,
  "arrow-up": ArrowUp,
  check: Check,
  "chevron-down": ChevronDown,
  diamond: Diamond,
  edit: SquarePen,
  grid: LayoutGrid,
  help: HelpCircle,
  menu: Menu,
  mic: Mic,
  more: MoreVertical,
  pencil: Pencil,
  pin: Pin,
  plus: Plus,
  search: Search,
  settings: Settings,
  share: Share2,
  trash: Trash2,
  warning: MessageSquareWarning,
};

interface RuntimeOption {
  id: string;
  name: string;
  desc: string;
}

const HISTORY_LIST = [
  { id: 1, title: "Bridge 连接与移动端任务", pinned: true, active: false },
  { id: 2, title: "Agent 执行链路设计", pinned: false, active: false },
  { id: 3, title: "Tauri App UI Development wi...", pinned: false, active: true },
  { id: 4, title: "移动端卡顿原因与优化建议", pinned: false, active: false },
  { id: 5, title: "AI Agent 手机端连接方案", pinned: false, active: false },
  { id: 6, title: "Bridge Runtime 参数说明", pinned: false, active: false },
];

function runtimeOptions(props: {
  runtimeLabel: string;
  status: StatusMessage;
  config: ConfigPayload | undefined;
}): RuntimeOption[] {
  const provider = props.config?.provider || props.config?.provider_type || "Bridge";
  const model = props.config?.model || "Runtime";

  return [
    {
      id: "bridge-runtime",
      name: "Bridge Runtime",
      desc: props.status.text,
    },
    {
      id: "agent-session",
      name: "Agent Session",
      desc: `${provider} / ${model}`,
    },
    {
      id: "native-driver",
      name: "Native Driver",
      desc: "桌面侧原子执行",
    },
  ];
}

interface ChatHeaderProps {
  runtimeLabel: string;
  config: ConfigPayload | undefined;
  status: StatusMessage;
  bridgeUrl: string;
  runtimeMenuOpen: boolean;
  onOpenSidebar: () => void;
  onToggleRuntimeMenu: () => void;
  onCloseRuntimeMenu: () => void;
  onOpenSettings: () => void;
  onOpenMoreMenu: () => void;
  onNewSession: () => void;
}

export function ChatHeader(props: ChatHeaderProps) {
  return (
    <header className="chat-header">
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
        <IconButton label="新任务" icon="edit" onClick={props.onNewSession} />
        <IconButton label="更多操作" icon="more" onClick={props.onOpenMoreMenu} />
      </div>
    </header>
  );
}

interface RuntimeMenuProps {
  config: ConfigPayload | undefined;
  status: StatusMessage;
  bridgeUrl: string;
  runtimeLabel: string;
  onClose: () => void;
  onOpenSettings: () => void;
}

function RuntimeMenu(props: RuntimeMenuProps) {
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

interface MobileSidebarProps {
  open: boolean;
  host: HostProfile | undefined;
  config: ConfigPayload | undefined;
  settings: StoredSettings;
  loading: boolean;
  sessionTitle: string;
  lastTraceId: string;
  onClose: () => void;
  onNewSession: () => void;
  onConnect: () => Promise<void>;
}

export function MobileSidebar(props: MobileSidebarProps) {
  const accountName = props.host?.productName || "Ghost-OS Mobile";
  const runtimeLabel = props.config?.model || props.config?.provider || props.config?.provider_type || "BRIDGE";
  const hasSession = Boolean(props.settings.sessionId || props.lastTraceId);

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
            <SidebarNavButton
              icon="grid"
              label={props.loading ? "连接中" : "连接 Bridge"}
              onClick={() => void props.onConnect()}
            />
            <SidebarNavButton icon="diamond" label="Agent" onClick={props.onClose} />
          </nav>

          <SidebarSection title="工作区">
            <SidebarNavButton icon="plus" label="新建工作区" onClick={props.onClose} />
          </SidebarSection>

          <SidebarSection title="最近">
            <div className="history-list">
              {HISTORY_LIST.map((item) => (
                <button
                  key={item.id}
                  className={`history-item ${item.active ? "is-active" : ""}`}
                  type="button"
                  title={item.id === 3 && hasSession ? props.sessionTitle : item.title}
                  onClick={props.onClose}
                >
                  <span>{item.id === 3 && hasSession ? props.sessionTitle : item.title}</span>
                  {item.pinned ? <UiIcon name="pin" /> : null}
                </button>
              ))}
            </div>
          </SidebarSection>
        </div>

        <footer className="sidebar-footer">
          <div className="sidebar-account">
            <div className="sidebar-avatar">G</div>
            <div className="sidebar-account-copy">
              <span>{accountName}</span>
              <strong>{runtimeLabel}</strong>
            </div>
          </div>
          <button
            className="sidebar-settings-button"
            type="button"
            aria-label="连接设置"
            title={props.settings.bridgeUrl}
            onClick={() => void props.onConnect()}
          >
            <UiIcon name="settings" />
          </button>
        </footer>
      </aside>
    </>
  );
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

interface AssistantIntroProps {
  host: HostProfile | undefined;
  config: ConfigPayload | undefined;
  lastTraceId: string;
  settings: StoredSettings;
  status: StatusMessage;
}

export function AssistantIntro(props: AssistantIntroProps) {
  const runtimeLabel =
    props.config?.provider && props.config.model
      ? `${props.config.provider} / ${props.config.model}`
      : props.config?.provider || props.config?.model || props.host?.target || "Bridge Runtime";
  const sessionLabel = props.settings.sessionId || props.lastTraceId || "当前任务";

  return (
    <AssistantPanel>
      <div className="assistant-copy">
        <p>
          Ghost-OS 移动端负责感知和交互，Bridge 负责状态、协议路由、编排和安全。手机端不直接执行桌面动作，而是把任务投递给桌面侧运行时。
        </p>
        <p>
          当前链路可以分为两类：<strong>连接状态</strong>（Bridge 是否可达、Runtime 是否读取成功）和{" "}
          <strong>任务状态</strong>（消息是否进入会话、trace_id 是否可追踪）。
        </p>
        <p>以下是移动端接入 Ghost-OS 时最关键的检查点：</p>

        <div className="assistant-section">
          <h3>1. 运行环境：Mobile UI vs Bridge Runtime</h3>
          <p>先确认手机端只承担入口职责，桌面 Bridge 才负责实际编排。</p>
          <ul>
            <li>
              <strong>当前 Runtime：</strong>
              {runtimeLabel}。
            </li>
            <li>
              <strong>当前会话：</strong>
              {sessionLabel}。
            </li>
          </ul>
        </div>
      </div>
    </AssistantPanel>
  );
}

interface AssistantReplyProps {
  reply: AgentPayload | undefined;
  status: StatusMessage;
  sessionId: string;
}

export function AssistantReply(props: AssistantReplyProps) {
  if (!props.reply && props.status.tone !== "error") {
    return null;
  }

  return (
    <AssistantPanel ariaLive="polite">
      <div className="assistant-copy">
        <p className={props.status.tone === "error" ? "error-text" : undefined}>
          {props.reply?.message || props.status.text}
        </p>
        {props.reply?.session_id || props.sessionId ? (
          <p className="assistant-meta">Session：{props.reply?.session_id || props.sessionId}</p>
        ) : null}
      </div>
    </AssistantPanel>
  );
}

export function ChatBubble(props: { children: ReactNode }) {
  return (
    <div className="message-row user-row">
      <div className="user-bubble">{props.children}</div>
    </div>
  );
}

function AssistantPanel(props: { children: ReactNode; ariaLive?: "polite" }) {
  return (
    <div className="message-row assistant-row">
      <section className="assistant-panel" aria-live={props.ariaLive}>
        {props.children}
      </section>
    </div>
  );
}

interface ChatComposerProps {
  value: string;
  disabled: boolean;
  loading: boolean;
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onChange: (value: string) => void;
  onOpenSettings: () => void;
}

const COMPOSER_TEXTAREA_MAX_HEIGHT_PX = 180;
const COMPOSER_TEXTAREA_EXPANDED_HEIGHT_PX = 48;

export function ChatComposer(props: ChatComposerProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [expanded, setExpanded] = useState(false);

  useAutosizeTextarea(textareaRef, props.value, setExpanded);

  return (
    <form className="composer-dock" onSubmit={props.onSubmit}>
      <div className="composer-shell">
        <div className={`composer-row ${expanded ? "is-expanded" : ""}`}>
          <IconButton label="添加任务上下文" icon="plus" variant="composer" onClick={props.onOpenSettings} />
          <textarea
            ref={textareaRef}
            value={props.value}
            rows={1}
            placeholder="问问 Ghost-OS"
            className="composer-input"
            onChange={(event) => props.onChange(event.currentTarget.value)}
            onInput={() => syncTextareaHeight(textareaRef.current, setExpanded)}
          />
          <div className="composer-actions">
            <IconButton label="语音输入" icon="mic" variant="composer" />
            <button
              className="send-button"
              type="submit"
              disabled={props.disabled}
              aria-busy={props.loading}
              aria-label="发送任务"
            >
              <UiIcon name="arrow-up" />
            </button>
          </div>
        </div>
      </div>
    </form>
  );
}

function useAutosizeTextarea(
  textareaRef: RefObject<HTMLTextAreaElement | null>,
  value: string,
  setExpanded: (expanded: boolean) => void,
) {
  useEffect(() => {
    syncTextareaHeight(textareaRef.current, setExpanded);
  }, [setExpanded, textareaRef, value]);
}

function syncTextareaHeight(
  textarea: HTMLTextAreaElement | null,
  setExpanded?: (expanded: boolean) => void,
) {
  if (!textarea) {
    return;
  }

  textarea.style.height = "auto";
  const nextHeight = Math.min(textarea.scrollHeight, COMPOSER_TEXTAREA_MAX_HEIGHT_PX);
  textarea.style.height = `${nextHeight}px`;
  setExpanded?.(nextHeight > COMPOSER_TEXTAREA_EXPANDED_HEIGHT_PX);
}

interface MoreActionSheetProps {
  open: boolean;
  hasTraceId: boolean;
  hasLocalConversation: boolean;
  onClose: () => void;
  onCopyTraceId: () => Promise<void>;
  onClearConversation: () => void;
}

export function MoreActionSheet(props: MoreActionSheetProps) {
  return (
    <>
      <button
        className={`sheet-scrim ${props.open ? "is-open" : ""}`}
        type="button"
        aria-label="关闭更多操作"
        onClick={props.onClose}
      />
      <div
        className={`action-sheet ${props.open ? "is-open" : ""}`}
        role="dialog"
        aria-hidden={!props.open}
        aria-modal="true"
        inert={props.open ? undefined : true}
      >
        <div className="sheet-handle" aria-hidden="true" />
        <div className="action-sheet-list">
          <ActionSheetButton
            icon="share"
            label="分享任务内容"
            onClick={() => {
              if (props.hasTraceId) {
                void props.onCopyTraceId();
                return;
              }
              props.onClose();
            }}
          />
          <ActionSheetButton icon="pin" label="固定" onClick={props.onClose} />
          <ActionSheetButton icon="pencil" label="重命名" onClick={props.onClose} />
          <ActionSheetButton icon="help" label="帮助" onClick={props.onClose} />
          <ActionSheetButton icon="warning" label="反馈" onClick={props.onClose} />
          <ActionSheetButton
            icon="trash"
            label="删除"
            disabled={!props.hasLocalConversation}
            onClick={props.onClearConversation}
          />
        </div>
      </div>
    </>
  );
}

function ActionSheetButton(props: { icon: UiIconName; label: string; disabled?: boolean; onClick: () => void }) {
  return (
    <button className="action-sheet-button" type="button" disabled={props.disabled} onClick={props.onClick}>
      <UiIcon name={props.icon} />
      <span>{props.label}</span>
    </button>
  );
}

interface IconButtonProps {
  label: string;
  icon: UiIconName;
  onClick?: () => void;
  disabled?: boolean;
  variant?: "header" | "composer";
}

function IconButton({ label, icon, onClick, disabled = false, variant = "header" }: IconButtonProps) {
  return (
    <button
      className={`icon-button icon-button-${variant}`}
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      title={label}
    >
      <UiIcon name={icon} />
    </button>
  );
}

function UiIcon(props: { name: UiIconName }) {
  const Icon = ICONS[props.name];

  return <Icon className={`ui-icon ui-icon-${props.name}`} aria-hidden="true" strokeWidth={1.75} />;
}
