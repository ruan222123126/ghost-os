import { parseSkillPayloadList } from './parser';
import type { SkillPayload } from '@/lib/types';

describe('lib/api/skills/parser', () => {
  it('parses skill payload list', () => {
    const expected: SkillPayload[] = [
      {
        id: 'skill_repo_release',
        name: 'release',
        description: 'repo release skill',
        path: '/tmp/project/.agents/skills/release',
        source: 'repo',
      },
      {
        id: 'skill_user_release',
        name: 'release',
        description: 'user release skill',
        path: '/tmp/home/.ghost-os/skills/release',
        source: 'user',
      },
    ];

    expect(parseSkillPayloadList(expected)).toEqual(expected);
  });

  it('rejects unknown fields in skill payloads', () => {
    expect(() => {
      parseSkillPayloadList([
        {
          id: 'skill_repo_release',
          name: 'release',
          description: 'repo release skill',
          path: '/tmp/project/.agents/skills/release',
          source: 'repo',
          unknown: 'ignored',
        },
      ]);
    }).toThrow('Invalid payload: unexpected field "unknown"');
  });

  it('throws for invalid payload source', () => {
    expect(() => {
      parseSkillPayloadList([
        {
          id: 'skill_invalid',
          name: 'invalid',
          description: 'invalid source',
          path: '/tmp/invalid',
          source: 'third_party',
        },
      ]);
    }).toThrow('skills list[0].source');
  });
});
