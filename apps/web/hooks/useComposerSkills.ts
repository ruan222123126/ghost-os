'use client';

import { useCallback, useState } from 'react';
import { listSkills } from '@/lib/api/skills/api';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SkillPayload } from '@/lib/types';

export function useComposerSkills() {
  const { copy } = useWebLocale();
  const [skills, setSkills] = useState<SkillPayload[]>([]);
  const [skillsLoading, setSkillsLoading] = useState(false);
  const [skillError, setSkillError] = useState('');

  const refreshSkills = useCallback(async () => {
    setSkillsLoading(true);
    try {
      const payload = await listSkills();
      setSkills(payload);
      setSkillError('');
    } catch (error) {
      setSkillError(toErrorMessage(error, copy.system.failedToLoadSkills));
    } finally {
      setSkillsLoading(false);
    }
  }, [copy.system.failedToLoadSkills]);

  return {
    skillError,
    skills,
    skillsLoading,
    refreshSkills,
  };
}
