import type { ChatSelectedSkill, ChatSendInput } from '@/lib/types';
import {
  buildSelectedSkillMessage,
  parseSelectedSkillMessage,
} from "../../shared/selectedSkillMessage";

interface ParsedSelectedSkillMessage {
  message: string;
  selectedSkill?: ChatSelectedSkill;
}

export function buildAgentMessageWithSelectedSkill(input: ChatSendInput): string {
  return buildSelectedSkillMessage(input);
}

export function parseAgentMessageWithSelectedSkill(raw: string): ParsedSelectedSkillMessage {
  return parseSelectedSkillMessage<ChatSelectedSkill>(raw);
}
