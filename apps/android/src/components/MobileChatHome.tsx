import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { FormEvent, ReactNode, RefObject } from "react";
import {
  ArrowDown,
  ArrowUp,
  Camera,
  Check,
  ChevronDown,
  Eye,
  FileText,
  HelpCircle,
  Image as ImageIcon,
  Lightbulb,
  Menu,
  MessageSquareWarning,
  Mic,
  MoreVertical,
  Pencil,
  Paperclip,
  Pin,
  Plus,
  Puzzle,
  Search,
  Settings,
  Share2,
  Sparkles,
  SquarePen,
  Trash2,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { AgentPayload, ConfigPayload, HostProfile, StatusMessage, StoredSettings } from "../mobileTypes";

type UiIconName =
  | "arrow-down"
  | "arrow-up"
  | "camera"
  | "check"
  | "chevron-down"
  | "edit"
  | "eye"
  | "file-text"
  | "help"
  | "image"
  | "lightbulb"
  | "menu"
  | "mic"
  | "more"
  | "paperclip"
  | "pencil"
  | "pin"
  | "plus"
  | "puzzle"
  | "search"
  | "settings"
  | "share"
  | "sparkles"
  | "trash"
  | "warning";

const ICONS: Record<UiIconName, LucideIcon> = {
  "arrow-down": ArrowDown,
  "arrow-up": ArrowUp,
  camera: Camera,
  check: Check,
  "chevron-down": ChevronDown,
  edit: SquarePen,
  eye: Eye,
  "file-text": FileText,
  help: HelpCircle,
  image: ImageIcon,
  lightbulb: Lightbulb,
  menu: Menu,
  mic: Mic,
  more: MoreVertical,
  paperclip: Paperclip,
  pencil: Pencil,
  pin: Pin,
  plus: Plus,
  puzzle: Puzzle,
  search: Search,
  settings: Settings,
  share: Share2,
  sparkles: Sparkles,
  trash: Trash2,
  warning: MessageSquareWarning,
};

interface RuntimeOption {
  id: string;
  name: string;
  desc: string;
}

const HISTORY_LIST = [
  { id: 1, title: "Bridge 连接与移动端任务", pinned: true },
  { id: 2, title: "Agent 执行链路设计", pinned: false },
  { id: 3, title: "Tauri App UI Development wi...", pinned: false },
  { id: 4, title: "移动端卡顿原因与优化建议", pinned: false },
  { id: 5, title: "AI Agent 手机端连接方案", pinned: false },
  { id: 6, title: "Bridge Runtime 参数说明", pinned: false },
];

const DRAWING_PLACEHOLDERS = [
  { id: 1, title: "绘画草稿 01", desc: "角色设定、姿态参考、画面比例" },
  { id: 2, title: "场景氛围板", desc: "光线、色彩、空间层次" },
  { id: 3, title: "移动端图标方案", desc: "线稿、填色、导出规格" },
  { id: 4, title: "产品概念图", desc: "三视图、材质、局部细节" },
  { id: 5, title: "启动页插画", desc: "主视觉、背景、留白区域" },
  { id: 6, title: "头像变体", desc: "表情、服饰、风格统一" },
  { id: 7, title: "工作流缩略图", desc: "节点、连线、状态标识" },
  { id: 8, title: "空状态插画", desc: "轻量占位、低对比背景" },
];

const CONVERSATION_PLACEHOLDER_SECTIONS = [
  {
    title: "1. 运行环境与首屏表现",
    body: "先确认当前是在开发模式还是生产构建中观察卡顿。开发模式会保留更多校验和热更新逻辑，滚动、输入和组件重渲染都可能更重。",
    bullets: ["记录首屏加载时间", "检查资源体积与请求数量", "确认是否存在重复初始化"],
  },
  {
    title: "2. 交互卡顿排查",
    body: "会话页最容易暴露输入框、滚动容器和长文本渲染的问题。这里用较长内容撑开页面，便于观察顶部栏、底部输入框和向下滚动按钮的层级关系。",
    bullets: ["滚动区域应独立于底部输入框", "长文本不能挤压按钮", "弹出菜单遮罩需要阻止底层滚动"],
  },
  {
    title: "3. 状态与协议边界",
    body: "移动端只负责感知和交互，不直接做核心编排。发送消息后保留 trace_id，便于把 UI 操作和 Bridge 响应串起来。",
    bullets: ["请求保持 trace_id", "错误直接展示", "新会话只清空本地会话状态"],
  },
  {
    title: "4. 后续内容占位",
    body: "这段内容用于验证上下滑动效果。真实接入后可以替换为流式回复、工具执行状态、引用卡片或任务步骤。",
    bullets: ["占位段落一", "占位段落二", "占位段落三"],
  },
];

const EMPTY_STATE_SUGGESTIONS = [
  { icon: "sparkles", text: "启动 Agent", tone: "blue" },
  { icon: "eye", text: "分析屏幕", tone: "purple" },
  { icon: "lightbulb", text: "制定执行计划", tone: "yellow" },
  { icon: "file-text", text: "总结会话", tone: "orange" },
] as const satisfies ReadonlyArray<{ icon: UiIconName; text: string; tone: "blue" | "orange" | "purple" | "yellow" }>;

const COMPOSER_MENU_OPTIONS = [
  { icon: "camera", label: "相机", unavailable: true },
  { icon: "image", label: "照片", unavailable: true },
  { icon: "paperclip", label: "文件", unavailable: true },
  { icon: "puzzle", label: "技能", unavailable: false },
] as const satisfies ReadonlyArray<{ icon: UiIconName; label: string; unavailable: boolean }>;

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
  onClose: () => void;
  onNewSession: () => void;
  onSelectSession: (title: string) => void;
  onConnect: () => Promise<void>;
}

export function MobileSidebar(props: MobileSidebarProps) {
  const accountName = props.host?.productName || "Ghost-OS Mobile";
  const isBridgeConnected = Boolean(props.config);
  const bridgeStatusLabel = isBridgeConnected ? "CONNECTED" : "DISCONNECTED";
  const [activeMode, setActiveMode] = useState<"chat" | "drawing">("chat");
  const [activeHistoryId, setActiveHistoryId] = useState(3);

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
                {HISTORY_LIST.map((item) => {
                  const isActive = activeHistoryId === item.id;

                  return (
                    <button
                      key={item.id}
                      className={`history-item ${isActive ? "is-active" : ""}`}
                      type="button"
                      title={item.title}
                      onClick={() => {
                        setActiveHistoryId(item.id);
                        props.onSelectSession(item.title);
                      }}
                    >
                      <span>{item.title}</span>
                      {item.pinned ? <UiIcon name="pin" /> : null}
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
  onSelectSuggestion: (value: string) => void;
}

export function AssistantIntro(props: AssistantIntroProps) {
  return (
    <div className="empty-state">
      <h2>有什么可以帮忙的？</h2>
      <div className="suggestion-grid">
        {EMPTY_STATE_SUGGESTIONS.map((suggestion) => (
          <SuggestionButton
            key={suggestion.text}
            icon={suggestion.icon}
            text={suggestion.text}
            tone={suggestion.tone}
            onClick={() => props.onSelectSuggestion(suggestion.text)}
          />
        ))}
      </div>
    </div>
  );
}

function SuggestionButton(props: {
  icon: UiIconName;
  text: string;
  tone: "blue" | "orange" | "purple" | "yellow";
  onClick: () => void;
}) {
  return (
    <button className="suggestion-button" type="button" onClick={props.onClick}>
      <span className={`suggestion-icon suggestion-icon-${props.tone}`}>
        <UiIcon name={props.icon} />
      </span>
      <span>{props.text}</span>
    </button>
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

export function ConversationPlaceholder() {
  return (
    <AssistantPanel>
      <div className="assistant-copy conversation-placeholder">
        <p>
          下面是会话中页面的占位内容，用来检查顶部栏、三点菜单、左侧抽屉和首页输入框在长内容下的滚动表现。
        </p>
        {CONVERSATION_PLACEHOLDER_SECTIONS.map((section) => (
          <section key={section.title} className="conversation-placeholder-section">
            <h3>{section.title}</h3>
            <p>{section.body}</p>
            <ul>
              {section.bullets.map((bullet) => (
                <li key={bullet}>{bullet}</li>
              ))}
            </ul>
          </section>
        ))}
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

const COMPOSER_TEXTAREA_COLLAPSED_HEIGHT_PX = 52;
const COMPOSER_TEXTAREA_MAX_HEIGHT_PX = 200;
const MULTILINE_TEXT_THRESHOLD = 30;

export function ChatComposer(props: ChatComposerProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const composerRef = useRef<HTMLFormElement>(null);
  const [focused, setFocused] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const hasValue = props.value.trim().length > 0;
  const isMultiLine = props.value.includes("\n") || props.value.length > MULTILINE_TEXT_THRESHOLD;

  useAutosizeTextarea(textareaRef, props.value, isMultiLine);
  useCloseComposerMenu(composerRef, menuOpen, setMenuOpen);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    setMenuOpen(false);
    await props.onSubmit(event);
  }

  function handleMenuOption(label: string, unavailable: boolean): void {
    if (unavailable) {
      return;
    }

    setMenuOpen(false);
    if (label === "技能") {
      props.onOpenSettings();
    }
  }

  return (
    <form ref={composerRef} className="composer-dock" onSubmit={(event) => void handleSubmit(event)}>
      {menuOpen ? (
        <div className="composer-attachment-menu" role="menu" aria-label="添加内容">
          {COMPOSER_MENU_OPTIONS.map((option) => (
            <button
              key={option.label}
              className="composer-menu-option"
              type="button"
              role="menuitem"
              disabled={option.unavailable}
              title={option.unavailable ? "暂未接入" : option.label}
              onClick={() => handleMenuOption(option.label, option.unavailable)}
            >
              <span className="composer-menu-icon">
                <UiIcon name={option.icon} />
              </span>
              <span>{option.label}</span>
            </button>
          ))}
        </div>
      ) : null}

      <div className={`composer-shell ${focused ? "is-focused" : ""} ${isMultiLine ? "is-multiline" : ""}`}>
        <div className="composer-textarea-wrap">
          <button
            className={`icon-button icon-button-composer composer-menu-trigger ${menuOpen ? "is-open" : ""}`}
            type="button"
            aria-label="添加内容"
            aria-expanded={menuOpen}
            aria-haspopup="menu"
            onClick={() => setMenuOpen((current) => !current)}
          >
            <UiIcon name="plus" />
          </button>
          <textarea
            ref={textareaRef}
            value={props.value}
            rows={1}
            placeholder="问问 Ghost-OS"
            className="composer-input"
            onChange={(event) => props.onChange(event.currentTarget.value)}
            onBlur={() => setFocused(false)}
            onFocus={() => setFocused(true)}
            onInput={() => syncTextareaHeight(textareaRef.current, isMultiLine)}
          />
          <div className="composer-actions">
            <IconButton label="语音输入" icon="mic" variant="composer" />
            <span className={`send-button-slot ${hasValue ? "is-visible" : ""}`} aria-hidden={!hasValue}>
              <button
                className="send-button"
                type="submit"
                disabled={props.disabled}
                aria-busy={props.loading}
                aria-label="发送任务"
                tabIndex={hasValue ? 0 : -1}
              >
                <UiIcon name="arrow-up" />
              </button>
            </span>
          </div>
        </div>
        <div className="composer-bottom-spacer" aria-hidden="true" />
      </div>
    </form>
  );
}

function useCloseComposerMenu(
  composerRef: RefObject<HTMLFormElement | null>,
  menuOpen: boolean,
  setMenuOpen: (open: boolean) => void,
) {
  useEffect(() => {
    if (!menuOpen) {
      return;
    }

    function handlePointerDown(event: PointerEvent): void {
      if (!(event.target instanceof Node) || composerRef.current?.contains(event.target)) {
        return;
      }

      setMenuOpen(false);
    }

    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, [composerRef, menuOpen, setMenuOpen]);
}

function useAutosizeTextarea(
  textareaRef: RefObject<HTMLTextAreaElement | null>,
  value: string,
  isMultiLine: boolean,
) {
  useLayoutEffect(() => {
    syncTextareaHeight(textareaRef.current, isMultiLine);
  }, [isMultiLine, textareaRef, value]);
}

function syncTextareaHeight(textarea: HTMLTextAreaElement | null, isMultiLine: boolean) {
  if (!textarea) {
    return;
  }

  textarea.style.transition = "none";
  textarea.style.height = "auto";
  const scrollHeight = textarea.scrollHeight;
  void textarea.offsetHeight;
  textarea.style.transition = "";
  textarea.style.height = isMultiLine
    ? `${Math.min(scrollHeight, COMPOSER_TEXTAREA_MAX_HEIGHT_PX)}px`
    : `${COMPOSER_TEXTAREA_COLLAPSED_HEIGHT_PX}px`;
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
            label="分享对话内容"
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
