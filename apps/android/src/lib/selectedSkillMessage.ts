import type { ChatSelectedSkill } from "../mobileTypes";
import {
  buildSelectedSkillMessage,
  parseSelectedSkillMessage,
} from "../../../shared/selectedSkillMessage";

interface BuildSelectedSkillInput {
  message: string;
  selectedSkill?: ChatSelectedSkill;
}

interface ParsedSelectedSkillMessage {
  message: string;
  selectedSkill?: ChatSelectedSkill;
}

export function buildAgentMessageWithSelectedSkill(input: BuildSelectedSkillInput): string {
  return buildSelectedSkillMessage(input);
}

export function parseAgentMessageWithSelectedSkill(raw: string): ParsedSelectedSkillMessage {
  return parseSelectedSkillMessage<ChatSelectedSkill>(raw);
}
