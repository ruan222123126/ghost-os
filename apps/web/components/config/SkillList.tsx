'use client';

import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SkillPayload } from '@/lib/types';

const SKILL_SKELETON_COUNT = 3;

interface SkillListProps {
  skills: SkillPayload[];
  loading: boolean;
  controlsDisabled: boolean;
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
  const { skills, loading, controlsDisabled, onDelete } = props;

  if (loading) {
    return (
      <div className="grid grid-cols-1 gap-3">
        {Array.from({ length: SKILL_SKELETON_COUNT }).map((_, index) => (
          <div
            key={`skill-skeleton-${index}`}
            className="h-[130px] animate-pulse rounded-[16px] border border-[#E5E5E5] bg-[#FAFAFA]"
          />
        ))}
      </div>
    );
  }

  if (skills.length === 0) {
    return (
      <div className="rounded-[16px] border border-[#E5E5E5] bg-white px-6 py-8 text-center text-[13px] text-[#737373]">
        {copy.settings.skillsNoItems}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3">
      {skills.map((skill) => (
        <article key={skill.id} className="group flex items-start justify-between gap-3 rounded-[16px] border border-[#E5E5E5] bg-white p-5 transition-all hover:border-[#111111]">
          <div className="min-w-0">
            <div className="mb-2 flex items-center gap-2">
              <span className="inline-flex rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
                {formatSource(skill.source, copy)}
              </span>
              <span className="font-mono text-[11px] text-[#737373]">{skill.id}</span>
            </div>
            <p className="mb-1 line-clamp-1 text-[14px] font-medium text-[#111111]">{skill.name}</p>
            <p className="mb-1 line-clamp-2 text-[13px] text-[#737373]">{skill.description}</p>
            <p className="truncate font-mono text-[12px] text-[#737373]">{skill.path}</p>
          </div>

          <div className="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
            <button
              type="button"
              disabled={controlsDisabled}
              onClick={() => requestSkillDelete({ id: skill.id, onDelete, message: copy.settings.skillsDeleteConfirm(skill.name) })}
              className="rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#FEF2F2] hover:text-[#DC2626] disabled:cursor-not-allowed disabled:opacity-50"
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

export function requestSkillDelete(input: SkillDeleteRequest) {
  const prompt = input.message ?? `Delete skill \"${input.name ?? input.id}\"? This will remove its files from disk.`;
  if (typeof window !== 'undefined' && !window.confirm(prompt)) {
    return;
  }
  ignorePromise(input.onDelete(input.id));
}
