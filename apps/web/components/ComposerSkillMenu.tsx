'use client';

import type { FC } from 'react';
import type { SkillPayload } from '@/lib/types';

interface ComposerSkillMenuProps {
  emptyLabel: string;
  error: string;
  loadingLabel: string;
  loading: boolean;
  onRefresh: () => void;
  onSelect: (skill: SkillPayload) => void;
  refreshLabel: string;
  skills: SkillPayload[];
  title: string;
}

export const ComposerSkillMenu: FC<ComposerSkillMenuProps> = ({
  emptyLabel,
  error,
  loadingLabel,
  loading,
  onRefresh,
  onSelect,
  refreshLabel,
  skills,
  title,
}) => {
  const enabledSkills = sortEnabledSkills(skills);

  return (
    <div className="composer-skill-menu" role="menu" aria-label={title}>
      <div className="composer-skill-menu-head">
        <span>{title}</span>
        <button type="button" className="composer-skill-refresh" onClick={onRefresh}>
          {refreshLabel}
        </button>
      </div>

      {loading ? <ComposerSkillStatus label={loadingLabel} /> : null}
      {!loading && error ? <ComposerSkillStatus label={error} tone="error" /> : null}
      {!loading && !error && enabledSkills.length === 0 ? <ComposerSkillStatus label={emptyLabel} /> : null}

      {!loading && !error && enabledSkills.length > 0 ? (
        <div className="composer-skill-list">
          {enabledSkills.map((skill) => (
            <button
              type="button"
              key={skill.id}
              className="composer-skill-item"
              role="menuitem"
              onClick={() => onSelect(skill)}
            >
              <span className="composer-skill-item-name">{skill.name}</span>
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
};

function ComposerSkillStatus({
  label,
  tone = 'neutral',
}: {
  label: string;
  tone?: 'error' | 'neutral';
}) {
  return (
    <div className={`composer-skill-status is-${tone}`} role="status">
      {label}
    </div>
  );
}

function sortEnabledSkills(skills: SkillPayload[]): SkillPayload[] {
  return skills
    .filter((skill) => skill.enabled)
    .sort((left, right) => {
      if (left.name !== right.name) {
        return left.name.localeCompare(right.name);
      }
      return left.source.localeCompare(right.source);
    });
}
