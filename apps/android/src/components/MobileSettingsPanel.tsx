import { useEffect, useState } from "react";
import { ArrowLeft, Globe, RefreshCw, Server, Sparkles, Workflow } from "lucide-react";
import type {
  ConfigPayload,
  ExternalCodexPermissionMode,
  OrchestrationTaskPayload,
  ProviderConfigInputPayload,
  ProviderListPayload,
  SkillPayload,
  StatusMessage,
  TaskPayload,
} from "../mobileTypes";
import { MobileOrchestrationSettings } from "./MobileOrchestrationSettings";
import { MobileProviderSettings } from "./MobileProviderSettings";
import { SettingsButton, SettingsSection } from "./MobileSettingsControls";
import { MobileSkillSettings } from "./MobileSkillSettings";
import { MobileTaskSettings } from "./MobileTaskSettings";
import "./MobileSettingsPanel.css";

interface MobileSettingsPanelProps {
  codexPermissionMode?: ExternalCodexPermissionMode;
  config: ConfigPayload | undefined;
  connectionStatus: StatusMessage;
  open: boolean;
  orchestrationList: OrchestrationTaskPayload[] | undefined;
  orchestrationListError: string;
  providerList: ProviderListPayload | undefined;
  runningTaskId: string;
  skillList: SkillPayload[] | undefined;
  skillListError: string;
  taskList: TaskPayload[] | undefined;
  taskListError: string;
  onActivateProvider: (name: string) => Promise<boolean>;
  onClose: () => void;
  onCreateProvider: (provider: ProviderConfigInputPayload) => Promise<boolean>;
  onDeleteOrchestration: (id: string) => Promise<boolean>;
  onDeleteProvider: (name: string) => Promise<boolean>;
  onDeleteSkill: (id: string) => Promise<boolean>;
  onDeleteTask: (id: string) => Promise<boolean>;
  onRefreshOrchestrations: () => Promise<boolean>;
  onRefreshProviders: () => Promise<boolean>;
  onRefreshSkills: () => Promise<boolean>;
  onRefreshTasks: () => Promise<boolean>;
  onRunTaskNow: (id: string, scope?: "user" | "orchestration") => Promise<boolean>;
  onSetOrchestrationEnabled: (id: string, enabled: boolean) => Promise<boolean>;
  onSetTaskEnabled: (id: string, enabled: boolean) => Promise<boolean>;
  onUpdateCodexPermission: (mode: ExternalCodexPermissionMode) => Promise<boolean>;
  onUpdateProvider: (name: string, provider: ProviderConfigInputPayload) => Promise<boolean>;
  onUpdateSkill: (id: string, enabled: boolean) => Promise<boolean>;
}

type SettingsView = "root" | "providers" | "skills" | "tasks" | "orchestrations";

export function MobileSettingsPanel(props: MobileSettingsPanelProps) {
  const [view, setView] = useState<SettingsView>("root");

  useEffect(() => {
    if (props.open) {
      setView("root");
    }
  }, [props.open]);

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
          {view === "providers" ? (
            <MobileProviderSettings
              config={props.config}
              providerList={props.providerList}
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
              runningTaskId={props.runningTaskId}
              onDeleteOrchestration={props.onDeleteOrchestration}
              onRefreshOrchestrations={props.onRefreshOrchestrations}
              onRunOrchestrationNow={(id) => props.onRunTaskNow(id, "orchestration")}
              onSetOrchestrationEnabled={props.onSetOrchestrationEnabled}
            />
          ) : (
            <SettingsRoot
              codexPermissionDisabled={props.connectionStatus.tone !== "success"}
              codexPermissionMode={props.codexPermissionMode ?? "default"}
              orchestrationDisabled={taskEntryDisabled(props.connectionStatus)}
              orchestrationSublabel={taskSublabel(
                props.connectionStatus,
                props.orchestrationList,
                props.orchestrationListError,
              )}
              providerDisabled={providerEntryDisabled(props.providerList)}
              providerSublabel={providerSublabel(props.config, props.providerList)}
              skillDisabled={skillEntryDisabled(props.connectionStatus)}
              skillSublabel={skillSublabel(props.connectionStatus, props.skillList, props.skillListError)}
              taskDisabled={taskEntryDisabled(props.connectionStatus)}
              taskSublabel={taskSublabel(props.connectionStatus, props.taskList, props.taskListError)}
              onOpenOrchestrations={() => setView("orchestrations")}
              onOpenProviders={() => setView("providers")}
              onOpenSkills={() => setView("skills")}
              onOpenTasks={() => setView("tasks")}
              onSetCodexPermission={props.onUpdateCodexPermission}
            />
          )}
        </div>
      </div>
    </section>
  );
}

function SettingsRoot(props: {
  codexPermissionDisabled: boolean;
  codexPermissionMode: ExternalCodexPermissionMode;
  orchestrationDisabled: boolean;
  orchestrationSublabel: string;
  providerDisabled: boolean;
  providerSublabel: string;
  skillDisabled: boolean;
  skillSublabel: string;
  taskDisabled: boolean;
  taskSublabel: string;
  onOpenOrchestrations: () => void;
  onOpenProviders: () => void;
  onOpenSkills: () => void;
  onOpenTasks: () => void;
  onSetCodexPermission: (mode: ExternalCodexPermissionMode) => Promise<boolean>;
}) {
  return (
    <>
      <SettingsSection title="管理">
        <div className="mobile-settings-card">
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

      <SettingsSection title="Agent">
        <div className="mobile-settings-card">
          <CodexPermissionSelector
            disabled={props.codexPermissionDisabled}
            mode={props.codexPermissionMode}
            onChange={props.onSetCodexPermission}
          />
        </div>
      </SettingsSection>
    </>
  );
}

function CodexPermissionSelector(props: {
  disabled: boolean;
  mode: ExternalCodexPermissionMode;
  onChange: (mode: ExternalCodexPermissionMode) => Promise<boolean>;
}) {
  const options: Array<{ label: string; mode: ExternalCodexPermissionMode }> = [
    { label: "只读", mode: "read-only" },
    { label: "默认", mode: "default" },
    { label: "自动重试", mode: "safe-yolo" },
    { label: "完全访问", mode: "yolo" },
  ];

  return (
    <div className="mobile-settings-codex-permission">
      <span className="mobile-settings-codex-permission-title">Codex 权限</span>
      <div className="mobile-settings-codex-permission-grid" role="group" aria-label="Codex 权限">
        {options.map((option) => (
          <button
            key={option.mode}
            type="button"
            className={props.mode === option.mode ? "is-active" : ""}
            disabled={props.disabled}
            aria-pressed={props.mode === option.mode}
            onClick={() => void props.onChange(option.mode)}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  );
}

function titleForView(view: SettingsView): string {
  switch (view) {
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
  providerList: ProviderListPayload | undefined,
): string {
  if (!providerList) {
    return "加载中";
  }
  const provider = config?.provider || providerList.active_provider || "";
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
