import { useState } from "react";
import { Power, RefreshCw, Trash2 } from "lucide-react";
import type { SkillPayload, SkillSource } from "../mobileTypes";
import "./MobileSkillSettings.css";

interface MobileSkillSettingsProps {
  loadError: string;
  skills: SkillPayload[] | undefined;
  onDeleteSkill: (id: string) => Promise<boolean>;
  onRefreshSkills: () => Promise<boolean>;
  onUpdateSkill: (id: string, enabled: boolean) => Promise<boolean>;
}

interface SkillCardProps {
  busy: boolean;
  skill: SkillPayload;
  onDelete: (skill: SkillPayload) => void;
  onToggle: (skill: SkillPayload) => void;
}

export function MobileSkillSettings(props: MobileSkillSettingsProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const displayError = error || props.loadError;
  const skills = sortSkillsForDisplay(props.skills ?? []);
  const enabledCount = skills.filter((skill) => skill.enabled).length;

  async function runSkillAction(action: () => Promise<boolean>, message: string): Promise<void> {
    setError("");
    setBusy(true);
    try {
      const ok = await action();
      if (!ok) {
        setError(message);
      }
    } finally {
      setBusy(false);
    }
  }

  function refreshSkills(): void {
    void runSkillAction(props.onRefreshSkills, "技能刷新失败");
  }

  function toggleSkill(skill: SkillPayload): void {
    void runSkillAction(() => props.onUpdateSkill(skill.id, !skill.enabled), "技能状态保存失败");
  }

  function deleteSkill(skill: SkillPayload): void {
    if (!window.confirm(`删除技能 ${skill.name}？其文件将从磁盘移除。`)) {
      return;
    }
    void runSkillAction(() => props.onDeleteSkill(skill.id), "技能删除失败");
  }

  return (
    <section className="mobile-settings-skill-shell" aria-labelledby="mobile-settings-skill-title">
      <div className="mobile-settings-skill-title-row">
        <div>
          <div className="mobile-settings-connection-title" id="mobile-settings-skill-title">
            技能
          </div>
          <p className="mobile-settings-skill-summary">{skillSummary(enabledCount, skills.length)}</p>
        </div>
        <button type="button" aria-label="刷新技能" disabled={busy} onClick={refreshSkills}>
          <RefreshCw className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
        </button>
      </div>

      {displayError ? <p className="mobile-settings-error is-card" role="alert">{displayError}</p> : null}

      {props.skills === undefined ? (
        <div className="mobile-settings-skill-empty-card">{props.loadError ? "技能未加载" : "技能加载中"}</div>
      ) : skills.length === 0 ? (
        <div className="mobile-settings-skill-empty-card">暂无技能</div>
      ) : (
        <div className="mobile-settings-skill-list">
          {skills.map((skill) => (
            <SkillCard
              key={skill.id}
              busy={busy}
              skill={skill}
              onDelete={deleteSkill}
              onToggle={toggleSkill}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function SkillCard(props: SkillCardProps) {
  const { busy, skill } = props;
  return (
    <article className={`mobile-settings-skill-card ${skill.enabled ? "is-enabled" : ""}`}>
      <div className="mobile-settings-skill-card-main">
        <div className="mobile-settings-skill-heading">
          <span>{skill.name}</span>
          <small>{skill.enabled ? "已启用" : "已停用"}</small>
        </div>
        <p>{skill.description || "无描述"}</p>
        <code>{skill.path}</code>
        <div className="mobile-settings-skill-meta">
          <span>{sourceLabel(skill.source)}</span>
          <span>{skill.id}</span>
        </div>
      </div>
      <div className="mobile-settings-skill-actions">
        <button type="button" disabled={busy} onClick={() => props.onToggle(skill)}>
          <Power className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          {skill.enabled ? "停用" : "启用"}
        </button>
        <button type="button" aria-label={`删除 ${skill.name}`} disabled={busy} onClick={() => props.onDelete(skill)}>
          <Trash2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
        </button>
      </div>
    </article>
  );
}

function skillSummary(enabledCount: number, totalCount: number): string {
  if (totalCount === 0) {
    return "未发现已安装技能";
  }
  return `${enabledCount}/${totalCount} 已启用`;
}

function sourceLabel(source: SkillSource): string {
  return source === "repo" ? "仓库" : "用户";
}

function sortSkillsForDisplay(skills: SkillPayload[]): SkillPayload[] {
  return [...skills].sort((left, right) => {
    if (left.enabled !== right.enabled) {
      return left.enabled ? -1 : 1;
    }
    if (left.name !== right.name) {
      return left.name.localeCompare(right.name);
    }
    return left.source.localeCompare(right.source);
  });
}
