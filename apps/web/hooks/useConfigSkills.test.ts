import {
  configSkillsReducer,
  initialConfigSkillsState,
} from './useConfigSkills';
import type { SkillPayload } from '@/lib/types';

describe('hooks/useConfigSkills reducer', () => {
  const sampleSkills: SkillPayload[] = [
    {
      id: 'skill_repo',
      name: 'release',
      description: 'repo skill',
      path: '/tmp/project/.agents/skills/release',
      source: 'repo',
      enabled: true,
    },
    {
      id: 'skill_user',
      name: 'release',
      description: 'user skill',
      path: '/tmp/home/.ghost-os/skills/release',
      source: 'user',
      enabled: false,
    },
  ];

  it('tracks loading state for refresh flow', () => {
    const loading = configSkillsReducer(initialConfigSkillsState, { type: 'load_start' });
    expect(loading.loading).toBe(true);

    const loaded = configSkillsReducer(loading, { type: 'load_success', skills: sampleSkills });
    expect(loaded.loading).toBe(false);
    expect(loaded.error).toBe('');
    expect(loaded.skills).toEqual(sampleSkills);
  });

  it('removes deleted skill immediately on delete success', () => {
    const baseState = {
      ...initialConfigSkillsState,
      skills: sampleSkills,
      saving: true,
    };

    const next = configSkillsReducer(baseState, { type: 'delete_success', id: 'skill_repo' });
    expect(next.saving).toBe(false);
    expect(next.skills).toEqual([sampleSkills[1]]);
  });

  it('stores explicit delete failure error state', () => {
    const saving = configSkillsReducer(initialConfigSkillsState, { type: 'delete_start' });
    expect(saving.saving).toBe(true);

    const failed = configSkillsReducer(saving, { type: 'delete_error', error: 'permission denied' });
    expect(failed.saving).toBe(false);
    expect(failed.error).toBe('permission denied');
  });

  it('replaces updated skill on save success', () => {
    const baseState = {
      ...initialConfigSkillsState,
      skills: sampleSkills,
      saving: true,
    };

    const next = configSkillsReducer(baseState, {
      type: 'update_success',
      skill: { ...sampleSkills[1], enabled: true },
    });
    expect(next.saving).toBe(false);
    expect(next.skills[1].enabled).toBe(true);
  });
});
