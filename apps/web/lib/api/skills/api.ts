import type { SkillPayload, SkillUpdateRequest } from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { parseSkillPayload, parseSkillPayloadList } from '@/lib/api/skills/parser';

export async function listSkills(): Promise<SkillPayload[]> {
  return requestJSON('/api/skills', {}, parseSkillPayloadList);
}

export async function deleteSkill(id: string): Promise<void> {
  await requestJSON(`/api/skills/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

export async function updateSkill(id: string, input: SkillUpdateRequest): Promise<SkillPayload> {
  return requestJSON(`/api/skills/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  }, parseSkillPayload);
}
