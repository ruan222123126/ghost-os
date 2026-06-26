import type { ComponentType, Dispatch, FormEvent, ReactNode, SetStateAction } from "react";
import { useEffect, useState } from "react";
import { ArrowLeft, Database, Globe, Key, Link2, RefreshCw, Server, Sparkles, Trash2, Wifi, Workflow } from "lucide-react";
import { deleteMobileCredential, saveMobileCredential } from "../lib/mobileCredentials";
import { hasTurnServer, parsePairingUri } from "../lib/mobileWebRTC";
import type {
  ConfigPayload,
  ProviderConfigInputPayload,
  ProviderListPayload,
  SkillPayload,
  StatusMessage,
  StoredSettings,
  OrchestrationTaskPayload,
  TaskPayload,
} from "../mobileTypes";
import { MobileOrchestrationSettings } from "./MobileOrchestrationSettings";
import { MobileTaskSettings } from "./MobileTaskSettings";
import { MobileProviderSettings } from "./MobileProviderSettings";
import { MobileSkillSettings } from "./MobileSkillSettings";
import "./MobileSettingsPanel.css";

interface MobileSettingsPanelProps {
  config: ConfigPayload | undefined;
  computerSessionPersistStatus: StatusMessage;
  connectionStatus: StatusMessage;
  localProviderList?: ProviderListPayload;
  open: boolean;
  providerList: ProviderListPayload | undefined;
  onActivateProvider: (name: string) => Promise<boolean>;
  onClose: () => void;
  onConnect: () => Promise<void>;
  onCreateProvider: (provider: ProviderConfigInputPayload) => Promise<boolean>;
  onDeleteOrchestration: (id: string) => Promise<boolean>;
  onDeleteProvider: (name: string) => Promise<boolean>;
  onDeleteSkill: (id: string) => Promise<boolean>;
  onDeleteTask: (id: string) => Promise<boolean>;
  onRefreshProviders: () => Promise<boolean>;
  onRefreshOrchestrations: () => Promise<boolean>;
  onRefreshSkills: () => Promise<boolean>;
  onRefreshTasks: () => Promise<boolean>;
  onRunTaskNow: (id: string) => Promise<boolean>;
  onSettingsChange: Dispatch<SetStateAction<StoredSettings>>;
  onSetTaskEnabled: (id: string, enabled: boolean) => Promise<boolean>;
  onSetOrchestrationEnabled: (id: string, enabled: boolean) => Promise<boolean>;
  onUpdateSkill: (id: string, enabled: boolean) => Promise<boolean>;
  onUpdateProvider: (name: string, provider: ProviderConfigInputPayload) => Promise<boolean>;
  runningTaskId: string;
  settings: StoredSettings;
  skillList: SkillPayload[] | undefined;
  skillListError: string;
  orchestrationList: OrchestrationTaskPayload[] | undefined;
  orchestrationListError: string;
  taskList: TaskPayload[] | undefined;
  taskListError: string;
}

type SettingsView = "root" | "connection" | "providers" | "skills" | "tasks" | "orchestrations";

export function MobileSettingsPanel(props: MobileSettingsPanelProps) {
  const [view, setView] = useState<SettingsView>("root");
  const [pairingUri, setPairingUri] = useState("");
  const [bridgeUrlDraft, setBridgeUrlDraft] = useState(props.settings.bridgeUrl);
  const [apiTokenDraft, setAPITokenDraft] = useState(props.settings.apiToken ?? "");
  const [pairingError, setPairingError] = useState("");
  const [pairingWarning, setPairingWarning] = useState("");

  useEffect(() => {
    if (props.open) {
      setView("root");
      setBridgeUrlDraft(props.settings.bridgeUrl);
      setAPITokenDraft(props.settings.apiToken ?? "");
      setPairingError("");
      setPairingWarning("");
    }
  }, [props.open]);

  async function importPairing(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setPairingError("");
    setPairingWarning("");
    try {
      const parsed = parsePairingUri(pairingUri);
      await saveMobileCredential(parsed.pairing.deviceId, parsed.secret);
      props.onSettingsChange((current) => ({
        ...current,
        connectionMode: "webrtc",
        pairing: parsed.pairing,
      }));
      setPairingUri("");
      if (!hasTurnServer(parsed.pairing)) {
        setPairingWarning("当前配对未包含 TURN，公网连接可能 ICE 失败");
      }
    } catch (error) {
      setPairingError(error instanceof Error ? error.message : String(error));
    }
  }

  async function removePairing(): Promise<void> {
    try {
      const deviceId = props.settings.pairing?.deviceId;
      if (deviceId) {
        await deleteMobileCredential(deviceId);
      }
      props.onSettingsChange((current) => ({
        ...current,
        lastSuccessfulConnection:
          current.lastSuccessfulConnection?.connectionMode === "webrtc"
          && current.lastSuccessfulConnection.pairing?.deviceId === deviceId
            ? undefined
            : current.lastSuccessfulConnection,
        pairing: undefined,
      }));
    } catch (error) {
      setPairingError(error instanceof Error ? error.message : String(error));
    }
  }

  function setAutoConnectEnabled(enabled: boolean): void {
    props.onSettingsChange((current) => ({ ...current, autoConnectEnabled: enabled }));
  }

  function setPersistComputerSessionsEnabled(enabled: boolean): void {
    props.onSettingsChange((current) => ({ ...current, persistComputerSessionsEnabled: enabled }));
  }

  function setRemoteExecutionEnabled(enabled: boolean): void {
    props.onSettingsChange((current) => ({ ...current, remoteExecutionEnabled: enabled }));
  }

  function setConnectionMode(mode: StoredSettings["connectionMode"]): void {
    props.onSettingsChange((current) => ({ ...current, connectionMode: mode }));
  }

  function saveBridgeURL(value: string): void {
    setBridgeUrlDraft(value);
    props.onSettingsChange((current) => ({ ...current, bridgeUrl: value }));
  }

  function saveAPIToken(value: string): void {
    setAPITokenDraft(value);
    props.onSettingsChange((current) => ({ ...current, apiToken: value }));
  }

  function handleBack(): void {
    if (view !== "root") {
      setView("root");
      return;
    }
    props.onClose();
  }

  return (
    <section
      className={`mobile-settings-panel ${props.open ? "is-open" : ""}`}
      role="dialog"
      aria-modal="true"
      aria-hidden={!props.open}
      aria-labelledby="mobile-settings-title"
      inert={props.open ? undefined : true}
    >
      <div className="mobile-settings-frame">
        <header className="mobile-settings-header">
          <button className="mobile-settings-back" type="button" aria-label="返回" onClick={handleBack}>
            <ArrowLeft className="mobile-settings-icon" aria-hidden="true" strokeWidth={2} />
          </button>
          <h1 id="mobile-settings-title">{titleForView(view)}</h1>
          <span className="mobile-settings-header-spacer" aria-hidden="true" />
        </header>

        <div className="mobile-settings-body">
          {view === "connection" ? (
            <ConnectionSettings
              apiTokenDraft={apiTokenDraft}
              bridgeUrlDraft={bridgeUrlDraft}
              pairingError={pairingError}
              pairingUri={pairingUri}
              pairingWarning={pairingWarning}
              connectionStatus={props.connectionStatus}
              settings={props.settings}
              onConnect={props.onConnect}
              onImportPairing={importPairing}
              onPairingUriChange={setPairingUri}
              onRemovePairing={removePairing}
              onSaveAPIToken={saveAPIToken}
              onSaveBridgeURL={saveBridgeURL}
              onSetConnectionMode={setConnectionMode}
            />
          ) : view === "providers" ? (
            <MobileProviderSettings
              config={props.config}
              providerList={props.localProviderList ?? props.providerList}
              onActivateProvider={props.onActivateProvider}
              onCreateProvider={props.onCreateProvider}
              onDeleteProvider={props.onDeleteProvider}
              onRefreshProviders={props.onRefreshProviders}
              onUpdateProvider={props.onUpdateProvider}
            />
          ) : view === "skills" ? (
            <MobileSkillSettings
              skills={props.skillList}
              loadError={props.skillListError}
              onDeleteSkill={props.onDeleteSkill}
              onRefreshSkills={props.onRefreshSkills}
              onUpdateSkill={props.onUpdateSkill}
            />
          ) : view === "tasks" ? (
            <MobileTaskSettings
              loadError={props.taskListError}
              runningTaskId={props.runningTaskId}
              tasks={props.taskList}
              onDeleteTask={props.onDeleteTask}
              onRefreshTasks={props.onRefreshTasks}
              onRunTaskNow={props.onRunTaskNow}
              onSetTaskEnabled={props.onSetTaskEnabled}
            />
          ) : view === "orchestrations" ? (
            <MobileOrchestrationSettings
              loadError={props.orchestrationListError}
              orchestrations={props.orchestrationList}
              onDeleteOrchestration={props.onDeleteOrchestration}
              onRefreshOrchestrations={props.onRefreshOrchestrations}
              onSetOrchestrationEnabled={props.onSetOrchestrationEnabled}
            />
          ) : (
            <SettingsRoot
              autoConnectEnabled={props.settings.autoConnectEnabled}
              connectionSublabel={connectionSublabel(props.settings)}
              orchestrationDisabled={taskEntryDisabled(props.connectionStatus)}
              orchestrationSublabel={taskSublabel(
                props.connectionStatus,
                props.orchestrationList,
                props.orchestrationListError,
              )}
              persistComputerSessionsEnabled={props.settings.persistComputerSessionsEnabled}
              persistComputerSessionsStatus={props.computerSessionPersistStatus}
              remoteExecutionEnabled={props.settings.remoteExecutionEnabled}
              remoteExecutionLocked={props.connectionStatus.tone !== "success"}
              taskDisabled={taskEntryDisabled(props.connectionStatus)}
              taskSublabel={taskSublabel(props.connectionStatus, props.taskList, props.taskListError)}
              providerDisabled={providerEntryDisabled(props.localProviderList)}
              providerSublabel={providerSublabel(props.config, props.connectionStatus, props.localProviderList)}
              skillDisabled={skillEntryDisabled(props.connectionStatus)}
              skillSublabel={skillSublabel(props.connectionStatus, props.skillList, props.skillListError)}
              onSetAutoConnectEnabled={setAutoConnectEnabled}
              onSetPersistComputerSessionsEnabled={setPersistComputerSessionsEnabled}
              onSetRemoteExecutionEnabled={setRemoteExecutionEnabled}
              onOpenConnection={() => setView("connection")}
              onOpenOrchestrations={() => setView("orchestrations")}
              onOpenTasks={() => setView("tasks")}
              onOpenProviders={() => setView("providers")}
              onOpenSkills={() => setView("skills")}
            />
          )}
        </div>
      </div>
    </section>
  );
}

function SettingsSection(props: { title: string; children: ReactNode }) {
  return (
    <section className="mobile-settings-section">
      <h2>{props.title}</h2>
      {props.children}
    </section>
  );
}

function SettingsButton(props: {
  icon: ComponentType<{ className?: string; "aria-hidden"?: true; strokeWidth?: number }>;
  disabled?: boolean;
  label: string;
  onClick?: () => void;
  sublabel?: string;
}) {
  const Icon = props.icon;

  return (
    <button className="mobile-settings-item-button" type="button" disabled={props.disabled} onClick={props.onClick}>
      <Icon className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
      <span className="mobile-settings-item-copy">
        <span>{props.label}</span>
        {props.sublabel ? <small>{props.sublabel}</small> : null}
      </span>
    </button>
  );
}

function SettingsRoot(props: {
  autoConnectEnabled: boolean;
  connectionSublabel: string;
  orchestrationDisabled: boolean;
  orchestrationSublabel: string;
  providerDisabled: boolean;
  providerSublabel: string;
  persistComputerSessionsEnabled: boolean;
  persistComputerSessionsStatus: StatusMessage;
  remoteExecutionEnabled: boolean;
  remoteExecutionLocked: boolean;
  skillDisabled: boolean;
  skillSublabel: string;
  taskDisabled: boolean;
  taskSublabel: string;
  onOpenConnection: () => void;
  onOpenOrchestrations: () => void;
  onOpenProviders: () => void;
  onOpenSkills: () => void;
  onOpenTasks: () => void;
  onSetAutoConnectEnabled: (enabled: boolean) => void;
  onSetPersistComputerSessionsEnabled: (enabled: boolean) => void;
  onSetRemoteExecutionEnabled: (enabled: boolean) => void;
}) {
  return (
    <>
      <SettingsSection title="连接">
        <div className="mobile-settings-card">
          <SettingsSwitchRow
            checked={props.autoConnectEnabled}
            icon={Wifi}
            label="是否自动连接"
            onChange={props.onSetAutoConnectEnabled}
          />
          <SettingsSwitchRow
            checked={props.persistComputerSessionsEnabled}
            icon={Database}
            label="是否持久化电脑会话内容"
            sublabel={props.persistComputerSessionsStatus.text}
            onChange={props.onSetPersistComputerSessionsEnabled}
          />
          <SettingsSwitchRow
            checked={props.remoteExecutionEnabled}
            disabled={props.remoteExecutionLocked}
            icon={Server}
            label="远程运行"
            sublabel={props.remoteExecutionLocked ? "仅连接成功后可开启" : "开启后跟随电脑端模型"}
            onChange={props.onSetRemoteExecutionEnabled}
          />
          <SettingsButton icon={Link2} label="连接" sublabel={props.connectionSublabel} onClick={props.onOpenConnection} />
          <SettingsButton
            icon={Server}
            label="供应商"
            sublabel={props.providerSublabel}
            disabled={props.providerDisabled}
            onClick={props.onOpenProviders}
          />
          <SettingsButton
            icon={Sparkles}
            label="技能"
            sublabel={props.skillSublabel}
            disabled={props.skillDisabled}
            onClick={props.onOpenSkills}
          />
          <SettingsButton
            icon={RefreshCw}
            label="任务"
            sublabel={props.taskSublabel}
            disabled={props.taskDisabled}
            onClick={props.onOpenTasks}
          />
          <SettingsButton
            icon={Workflow}
            label="编排"
            sublabel={props.orchestrationSublabel}
            disabled={props.orchestrationDisabled}
            onClick={props.onOpenOrchestrations}
          />
        </div>
      </SettingsSection>

      <SettingsSection title="应用">
        <div className="mobile-settings-card">
          <SettingsButton icon={Globe} label="语言" sublabel="中文" />
        </div>
      </SettingsSection>
    </>
  );
}

function SettingsSwitchRow(props: {
  checked: boolean;
  icon: ComponentType<{ className?: string; "aria-hidden"?: true; strokeWidth?: number }>;
  label: string;
  onChange: (checked: boolean) => void;
  sublabel?: string;
  disabled?: boolean;
}) {
  const Icon = props.icon;

  return (
    <button
      className="mobile-settings-auto-connect-row"
      type="button"
      role="switch"
      aria-checked={props.checked}
      disabled={props.disabled}
      onClick={() => props.onChange(!props.checked)}
    >
      <span className="mobile-settings-auto-connect-copy">
        <Icon className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
        <span>
          <span>{props.label}</span>
          {props.sublabel ? <small>{props.sublabel}</small> : null}
        </span>
      </span>
      <span className="mobile-settings-toggle-track" aria-hidden={true}>
        <span className="mobile-settings-toggle-thumb" />
      </span>
    </button>
  );
}

function titleForView(view: SettingsView): string {
  switch (view) {
    case "connection":
      return "连接";
    case "providers":
      return "供应商";
    case "skills":
      return "技能";
    case "tasks":
      return "任务";
    case "orchestrations":
      return "编排";
    case "root":
      return "设置";
  }
}

interface ConnectionSettingsProps {
  apiTokenDraft: string;
  bridgeUrlDraft: string;
  connectionStatus: StatusMessage;
  pairingError: string;
  pairingUri: string;
  pairingWarning: string;
  settings: StoredSettings;
  onConnect: () => Promise<void>;
  onImportPairing: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onPairingUriChange: (value: string) => void;
  onRemovePairing: () => Promise<void>;
  onSaveAPIToken: (value: string) => void;
  onSaveBridgeURL: (value: string) => void;
  onSetConnectionMode: (mode: StoredSettings["connectionMode"]) => void;
}

function ConnectionSettings(props: ConnectionSettingsProps) {
  const isWebRTC = props.settings.connectionMode === "webrtc";
  const isConnecting = props.connectionStatus.tone === "loading";
  const isConnected = props.connectionStatus.tone === "success";
  const connectButtonText = isConnecting ? "连接中" : isConnected ? "已连接" : "连接 Bridge";

  return (
    <section className="mobile-settings-connection-shell" aria-labelledby="mobile-settings-connection-title">
      <div className="mobile-settings-connection-title" id="mobile-settings-connection-title">
        连接
      </div>

      <div className="mobile-settings-connection-card">
        <div className="mobile-settings-mode-switch" role="group" aria-label="连接模式">
          <button
            className={isWebRTC ? "is-active" : ""}
            type="button"
            onClick={() => props.onSetConnectionMode("webrtc")}
            aria-pressed={isWebRTC}
          >
            <WebRTCSignalIcon />
            WebRTC
          </button>
          <button
            className={!isWebRTC ? "is-active" : ""}
            type="button"
            onClick={() => props.onSetConnectionMode("http")}
            aria-pressed={!isWebRTC}
          >
            <Server className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
            HTTP
          </button>
        </div>

        <div className="mobile-settings-connection-content">
          {isWebRTC ? (
            <div className="mobile-settings-connection-panel">
              <SettingsStaticRow
                icon={Wifi}
                label="WebRTC"
                sublabel={props.settings.pairing ? props.settings.pairing.pcId : "未配对"}
              />
              <form className="mobile-settings-pair-form" onSubmit={(event) => void props.onImportPairing(event)}>
                <div className="mobile-settings-textarea-wrap">
                  <textarea
                    value={props.pairingUri}
                    rows={3}
                    aria-label="WebRTC 配对 URI"
                    placeholder="ghost-os://mobile-pair?..."
                    onChange={(event) => props.onPairingUriChange(event.target.value)}
                  />
                  <span className="mobile-settings-resize-mark" aria-hidden={true}>
                    <span />
                    <span />
                  </span>
                </div>
                <button type="submit" disabled={!props.pairingUri.trim()}>
                  <Key className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
                  导入配对
                </button>
              </form>
              {props.pairingWarning ? <p className="mobile-settings-warning">{props.pairingWarning}</p> : null}
              {props.pairingError ? <p className="mobile-settings-error">{props.pairingError}</p> : null}
              {props.settings.pairing ? (
                <button
                  className="mobile-settings-danger-button"
                  type="button"
                  onClick={() => void props.onRemovePairing()}
                >
                  <Trash2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
                  删除配对
                </button>
              ) : null}
            </div>
          ) : (
            <div className="mobile-settings-connection-panel">
              <SettingsStaticRow icon={Server} label="HTTP" sublabel={props.bridgeUrlDraft || "未配置"} />
              <label className="mobile-settings-url-field">
                <span>Bridge URL</span>
                <input value={props.bridgeUrlDraft} onChange={(event) => props.onSaveBridgeURL(event.target.value)} />
              </label>
              <label className="mobile-settings-url-field mobile-settings-api-token-field">
                <span>API Token</span>
                <input
                  value={props.apiTokenDraft}
                  type="password"
                  autoCapitalize="none"
                  autoComplete="off"
                  spellCheck={false}
                  onChange={(event) => props.onSaveAPIToken(event.target.value)}
                />
              </label>
            </div>
          )}

          <ConnectionStatusMessage status={props.connectionStatus} />

          <button
            className={`mobile-settings-connect-button ${isConnected ? "is-connected" : ""}`}
            type="button"
            disabled={isConnecting}
            onClick={() => void props.onConnect()}
          >
            <Link2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
            {connectButtonText}
          </button>
        </div>
      </div>
    </section>
  );
}

function ConnectionStatusMessage(props: { status: StatusMessage }) {
  if (props.status.tone === "idle") {
    return null;
  }

  const role = props.status.tone === "error" ? "alert" : "status";

  return (
    <p className={`mobile-settings-connection-status is-${props.status.tone}`} role={role}>
      <span className="mobile-settings-status-dot" aria-hidden={true} />
      <span>{props.status.text}</span>
    </p>
  );
}

function WebRTCSignalIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="mobile-settings-icon"
      aria-hidden={true}
    >
      <circle cx="12" cy="12" r="2" />
      <path d="M16 8.5a5 5 0 0 1 0 7" />
      <path d="M8 8.5a5 5 0 0 0 0 7" />
    </svg>
  );
}

function connectionSublabel(settings: StoredSettings): string {
  if (settings.connectionMode === "http") {
    return "HTTP fallback";
  }
  return settings.pairing ? `WebRTC / ${settings.pairing.pcId}` : "WebRTC / 未配对";
}

function providerEntryDisabled(providerList: ProviderListPayload | undefined): boolean {
  return !providerList;
}

function skillEntryDisabled(connectionStatus: StatusMessage): boolean {
  return connectionStatus.tone !== "success";
}

function taskEntryDisabled(connectionStatus: StatusMessage): boolean {
  return connectionStatus.tone !== "success";
}

function providerSublabel(
  config: ConfigPayload | undefined,
  _connectionStatus: StatusMessage,
  providerList: ProviderListPayload | undefined,
): string {
  if (!providerList) {
    return "加载中";
  }
  const provider = providerList.active_provider || config?.provider || "";
  if (provider && config?.model) {
    return `${provider} / ${config.model}`;
  }
  return provider || "未激活";
}

function skillSublabel(connectionStatus: StatusMessage, skillList: SkillPayload[] | undefined, error: string): string {
  if (connectionStatus.tone !== "success") {
    return "未连接";
  }
  if (error) {
    return error;
  }
  if (!skillList) {
    return "加载中";
  }
  const enabledCount = skillList.filter((skill) => skill.enabled).length;
  return `${enabledCount}/${skillList.length} 已启用`;
}

function taskSublabel<TTask extends { enabled: boolean }>(
  connectionStatus: StatusMessage,
  taskList: TTask[] | undefined,
  error: string,
): string {
  if (connectionStatus.tone !== "success") {
    return "未连接";
  }
  if (error) {
    return error;
  }
  if (!taskList) {
    return "加载中";
  }
  const enabledCount = taskList.filter((task) => task.enabled).length;
  return `${enabledCount}/${taskList.length} 已启用`;
}

function SettingsStaticRow(props: {
  icon: ComponentType<{ className?: string; "aria-hidden"?: true; strokeWidth?: number }>;
  label: string;
  sublabel?: string;
}) {
  const Icon = props.icon;

  return (
    <div className="mobile-settings-static-row">
      <Icon className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
      <span className="mobile-settings-item-copy">
        <span>{props.label}</span>
        {props.sublabel ? <small>{props.sublabel}</small> : null}
      </span>
    </div>
  );
}
