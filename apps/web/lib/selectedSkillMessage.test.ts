import {
  buildAgentMessageWithSelectedSkill,
  parseAgentMessageWithSelectedSkill,
} from './selectedSkillMessage';

describe('lib/selectedSkillMessage', () => {
  it('wraps and parses selected skill metadata without exposing it as user text', () => {
    const wrapped = buildAgentMessageWithSelectedSkill({
      images: [],
      message: 'ship the release',
      selectedSkill: {
        id: 'skill_release',
        name: 'release_flow',
      },
    });

    expect(wrapped).toContain('[Ghost-OS selected skill]');
    expect(wrapped).toContain('already been loaded through sfind');
    expect(wrapped).toContain('do not call sfind just to load or verify it');

    expect(parseAgentMessageWithSelectedSkill(wrapped)).toEqual({
      message: 'ship the release',
      selectedSkill: {
        id: 'skill_release',
        name: 'release_flow',
      },
    });
  });

  it('leaves regular user text unchanged', () => {
    expect(parseAgentMessageWithSelectedSkill('plain request')).toEqual({
      message: 'plain request',
    });
  });
});
