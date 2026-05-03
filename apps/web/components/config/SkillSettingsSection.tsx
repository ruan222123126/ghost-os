'use client';

import { SkillList } from '@/components/config/SkillList';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SkillPayload } from '@/lib/types';

interface SkillSettingsSectionProps {
  skills: SkillPayload[];
  loading: boolean;
  saving: boolean;
  onRefresh: () => Promise<void>;
  onUpdate: (id: string, enabled: boolean) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

export function SkillSettingsSection(props: SkillSettingsSectionProps) {
  const { copy } = useWebLocale();
  const { skills, loading, saving, onRefresh, onUpdate, onDelete } = props;
  const controlsDisabled = loading || saving;

  return (
    <section>
      <header className="mb-10 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.skillsTitle}</h1>
          <p className="text-[14px] text-[#737373]">{copy.settings.skillsDescription}</p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={() => {
              ignorePromise(onRefresh());
            }}
            className="rounded-full border border-[#E5E5E5] px-4 py-2 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.refresh}
          </button>
        </div>
      </header>

      <SkillList
        skills={skills}
        loading={loading}
        controlsDisabled={controlsDisabled}
        onUpdate={onUpdate}
        onDelete={onDelete}
      />
    </section>
  );
}
