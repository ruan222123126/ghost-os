import type { Dispatch, FormEvent, ReactNode, RefObject, SetStateAction } from "react";
import type { AgentPayload, ConfigPayload, HostProfile, StatusMessage, StoredSettings } from "../mobileTypes";

interface ChatHeaderProps {
  runtimeLabel: string;
  status: StatusMessage;
  onOpenSettings: () => void;
  onNewSession: () => void;
}

export function ChatHeader(props: ChatHeaderProps) {
  return (
    <header className="chat-header">
      <div className="header-left">
        <IconButton label="连接设置" icon="menu" onClick={props.onOpenSettings} />
        <div className="runtime-selector" title={props.runtimeLabel}>
          <span>{props.runtimeLabel}</span>
          <span className="ui-icon ui-icon-chevron-down" aria-hidden="true" />
        </div>
      </div>
      <div className="header-right">
        <span className={`status-pill status-${props.status.tone}`}>{props.status.text}</span>
        <IconButton label="新会话" icon="edit" onClick={props.onNewSession} />
        <IconButton label="连接设置" icon="more" onClick={props.onOpenSettings} />
      </div>
    </header>
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
  return (
    <AssistantPanel>
      <div className="assistant-copy">
        <p>
          Ghost-OS 是 AI 驱动的数字孪生执行层。移动端负责感知和交互，Bridge 负责状态、协议路由、编排和安全。
        </p>
        <p>
          请求会通过标准消息总线传递，并保留 <code>trace_id</code>，便于把一次任务从手机、Bridge 到 native driver
          全链路定位。
        </p>
        <p>当前首页聚焦一个闭环：连接 Bridge、发送任务、查看回复。</p>
      </div>

      <RuntimeGrid
        host={props.host}
        config={props.config}
        lastTraceId={props.lastTraceId}
        sessionId={props.settings.sessionId}
        status={props.status}
      />
    </AssistantPanel>
  );
}

interface RuntimeGridProps {
  host: HostProfile | undefined;
  config: ConfigPayload | undefined;
  lastTraceId: string;
  sessionId: string;
  status: StatusMessage;
}

function RuntimeGrid(props: RuntimeGridProps) {
  const hostLabel = props.host ? `${props.host.target}${props.host.mobile ? " mobile" : ""}` : "Tauri";

  return (
    <div className="runtime-grid" aria-label="运行状态">
      <Metric label="Host" value={hostLabel} />
      <Metric label="Provider" value={props.config?.provider || props.config?.provider_type || "-"} />
      <Metric label="Model" value={props.config?.model || "-"} />
      <Metric label="Trace" value={props.lastTraceId || "-"} />
      <Metric label="Session" value={props.sessionId || "-"} />
      <Metric label="Status" value={props.status.text} />
    </div>
  );
}

function Metric(props: { label: string; value: string }) {
  return (
    <div className="metric">
      <span>{props.label}</span>
      <strong>{props.value}</strong>
    </div>
  );
}

interface BridgeSettingsProps {
  refTarget: RefObject<HTMLElement | null>;
  settings: StoredSettings;
  apiToken: string;
  loading: boolean;
  onConnect: () => Promise<void>;
  onApiTokenChange: (value: string) => void;
  onSettingsChange: Dispatch<SetStateAction<StoredSettings>>;
}

export function BridgeSettings(props: BridgeSettingsProps) {
  return (
    <section ref={props.refTarget} className="settings-panel" aria-label="Bridge 连接参数">
      <div className="settings-panel-head">
        <div>
          <span className="section-kicker">Bridge</span>
          <h2>连接参数</h2>
        </div>
        <button className="connect-button" type="button" onClick={props.onConnect} disabled={props.loading}>
          {props.loading ? "连接中" : "连接"}
        </button>
      </div>

      <div className="settings-grid">
        <label className="field bridge-url-field">
          <span>Bridge URL</span>
          <input
            value={props.settings.bridgeUrl}
            inputMode="url"
            spellCheck={false}
            onChange={(event) =>
              props.onSettingsChange((current) => ({
                ...current,
                bridgeUrl: event.currentTarget.value,
              }))
            }
          />
        </label>
        <label className="field">
          <span>API Token</span>
          <input
            value={props.apiToken}
            type="password"
            autoComplete="off"
            onChange={(event) => props.onApiTokenChange(event.currentTarget.value)}
          />
        </label>
        <label className="field">
          <span>Session ID</span>
          <input
            value={props.settings.sessionId}
            spellCheck={false}
            onChange={(event) =>
              props.onSettingsChange((current) => ({
                ...current,
                sessionId: event.currentTarget.value,
              }))
            }
          />
        </label>
      </div>
    </section>
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
      <div className="response-header">
        <span>Assistant</span>
        <span>{props.reply?.session_id || props.sessionId || "-"}</span>
      </div>
      <p className={props.status.tone === "error" ? "error-text" : undefined}>
        {props.reply?.message || props.status.text}
      </p>
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

export function ChatComposer(props: ChatComposerProps) {
  return (
    <form className="composer-dock" onSubmit={props.onSubmit}>
      <div className="composer-shell">
        <IconButton label="连接设置" icon="plus" variant="composer" onClick={props.onOpenSettings} />
        <input
          value={props.value}
          type="text"
          placeholder="让 Ghost-OS 执行任务"
          className="composer-input"
          onChange={(event) => props.onChange(event.currentTarget.value)}
        />
        <IconButton label="语音输入未接入" icon="mic" variant="composer" disabled />
        <button className="send-button" type="submit" disabled={props.disabled} aria-label="发送任务">
          <span className={`ui-icon ${props.loading ? "ui-icon-audio-lines" : "ui-icon-send"}`} aria-hidden="true" />
        </button>
      </div>
    </form>
  );
}

interface IconButtonProps {
  label: string;
  icon: "menu" | "edit" | "more" | "plus" | "mic";
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
      <span className={`ui-icon ui-icon-${icon}`} aria-hidden="true" />
    </button>
  );
}
