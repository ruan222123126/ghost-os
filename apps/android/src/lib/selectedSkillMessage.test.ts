import { describe, expect, it } from "vitest";
import { buildAgentMessageWithSelectedSkill, parseAgentMessageWithSelectedSkill } from "./selectedSkillMessage";

describe("selectedSkillMessage", () => {
  it("wraps and parses selected skill metadata without exposing it as user text", () => {
    const wrapped = buildAgentMessageWithSelectedSkill({
      message: "发布版本",
      selectedSkill: {
        id: "skill_release",
        name: "release_flow",
      },
    });

    expect(wrapped).toContain("[Ghost-OS selected skill]");
    expect(parseAgentMessageWithSelectedSkill(wrapped)).toEqual({
      message: "发布版本",
      selectedSkill: {
        id: "skill_release",
        name: "release_flow",
      },
    });
  });

  it("leaves plain messages unchanged", () => {
    expect(parseAgentMessageWithSelectedSkill("plain request")).toEqual({
      message: "plain request",
    });
  });
});
