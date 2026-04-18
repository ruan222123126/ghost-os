import { deleteSkill, listSkills } from './api';
import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';
import type { SkillPayload } from '@/lib/types';

describe('lib/api/skills/api', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('listSkills calls GET /api/skills', async () => {
    const expected: SkillPayload[] = [
      {
        id: 'skill_repo_release',
        name: 'release',
        description: 'repo release skill',
        path: '/tmp/project/.agents/skills/release',
        source: 'repo',
      },
    ];
    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const skills = await listSkills();
    expect(skills).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/skills', expect.any(Object));
  });

  it('deleteSkill calls DELETE /api/skills/:id', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { deleted: true },
      error: '',
    });

    await deleteSkill('skill_repo_release');
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/skills/skill_repo_release',
      expect.objectContaining({ method: 'DELETE' }),
    );
  });
});
