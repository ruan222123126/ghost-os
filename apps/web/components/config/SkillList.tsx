'use client';

import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SkillPayload } from '@/lib/types';

const SKILL_SKELETON_COUNT = 3;

interface SkillListProps {
  skills: SkillPayload[];
  loading: boolean;
  controlsDisabled: boolean;
  onUpdate: (id: string, enabled: boolean) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

interface SkillDeleteRequest {
  id: string;
  name?: string;
  message?: string;
  onDelete: (id: string) => Promise<void>;
}

export function SkillList(props: SkillListProps) {
  const { copy } = useWebLocale();
  const { skills, loading, controlsDisabled, onUpdate, onDelete } = props;

  if (loading) {
    return (
      <div className="settings-card-list">
        {Array.from({ length: SKILL_SKELETON_COUNT }).map((_, index) => (
          <div
            key={`skill-skeleton-${index}`}
            className="settings-card-skeleton animate-pulse"
          />
        ))}
      </div>
    );
  }

  if (skills.length === 0) {
    return (
      <div className="settings-list-empty">
        {copy.settings.skillsNoItems}
      </div>
    );
  }

  return (
    <div className="settings-card-list">
      {sortSkillsForDisplay(skills).map((skill) => (
        <article key={skill.id} className="settings-config-card">
          <div className="settings-config-card-main">
            <div className="settings-card-badges">
              <span className="settings-card-badge">
                {formatSource(skill.source, copy)}
              </span>
              <span className="settings-card-badge">
                {skill.enabled ? copy.settings.enabled : copy.settings.disabled}
              </span>
              <span className="settings-card-id">{skill.id}</span>
            </div>
            <p className="settings-card-title is-strong line-clamp-1">{skill.name}</p>
            <p className="settings-card-meta line-clamp-2">{skill.description}</p>
            <p className="settings-card-meta is-mono truncate">{skill.path}</p>
          </div>

          <div className="settings-config-card-actions">
            <button
              type="button"
              disabled={controlsDisabled}
              onClick={() => {
                ignorePromise(onUpdate(skill.id, !skill.enabled));
              }}
              className="settings-card-action whitespace-nowrap"
            >
              {skill.enabled ? copy.settings.skillsDisable : copy.settings.skillsEnable}
            </button>
            <button
              type="button"
              disabled={controlsDisabled}
              onClick={() => requestSkillDelete({ id: skill.id, onDelete, message: copy.settings.skillsDeleteConfirm(skill.name) })}
              className="settings-card-action is-danger whitespace-nowrap"
            >
              {copy.settings.skillsDelete}
            </button>
          </div>
        </article>
      ))}
    </div>
  );
}

function formatSource(
  source: SkillPayload['source'],
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  return source === 'repo' ? copy.settings.skillsSourceRepo : copy.settings.skillsSourceUser;
}

export function sortSkillsForDisplay(skills: SkillPayload[]): SkillPayload[] {
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

export function requestSkillDelete(input: SkillDeleteRequest) {
  const prompt = input.message ?? `Delete skill \"${input.name ?? input.id}\"? This will remove its files from disk.`;
  if (typeof window !== 'undefined' && !window.confirm(prompt)) {
    return;
  }
  ignorePromise(input.onDelete(input.id));
}
