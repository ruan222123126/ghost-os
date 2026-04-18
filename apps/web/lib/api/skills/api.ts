import type { SkillPayload } from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { parseSkillPayloadList } from '@/lib/api/skills/parser';

export async function listSkills(): Promise<SkillPayload[]> {
  return requestJSON('/api/skills', {}, parseSkillPayloadList);
}

export async function deleteSkill(id: string): Promise<void> {
  await requestJSON(`/api/skills/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}
