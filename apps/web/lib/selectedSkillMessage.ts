import type { ChatSelectedSkill, ChatSendInput } from '@/lib/types';

const SELECTED_SKILL_HEADER = '[Ghost-OS selected skill]';
const USER_REQUEST_HEADER = '[User request]';

interface ParsedSelectedSkillMessage {
  message: string;
  selectedSkill?: ChatSelectedSkill;
}

export function buildAgentMessageWithSelectedSkill(input: ChatSendInput): string {
  const message = input.message.trim();
  const selectedSkill = input.selectedSkill;
  if (!selectedSkill) {
    return message;
  }

  return [
    SELECTED_SKILL_HEADER,
    JSON.stringify({
      id: selectedSkill.id,
      name: selectedSkill.name,
    }),
    'Before answering, call sfind with action "load" and skill_names containing exactly the selected skill name from the JSON metadata. Then use the loaded skill for the user request. If the user request is empty, ask what task should be done with this skill.',
    '',
    USER_REQUEST_HEADER,
    message,
  ].join('\n');
}

export function parseAgentMessageWithSelectedSkill(raw: string): ParsedSelectedSkillMessage {
  if (!raw.startsWith(`${SELECTED_SKILL_HEADER}\n`)) {
    return { message: raw };
  }

  const marker = `\n${USER_REQUEST_HEADER}\n`;
  const markerIndex = raw.indexOf(marker);
  if (markerIndex < 0) {
    return { message: raw };
  }

  const metadata = parseSelectedSkillMetadata(raw.slice(SELECTED_SKILL_HEADER.length, markerIndex).trimStart());
  if (!metadata) {
    return { message: raw };
  }

  return {
    message: raw.slice(markerIndex + marker.length).trim(),
    selectedSkill: metadata,
  };
}

function parseSelectedSkillMetadata(raw: string): ChatSelectedSkill | null {
  const firstLineEnd = raw.indexOf('\n');
  const metadataLine = (firstLineEnd >= 0 ? raw.slice(0, firstLineEnd) : raw).trim();
  if (!metadataLine) {
    return null;
  }

  try {
    const value = JSON.parse(metadataLine) as Record<string, unknown>;
    if (typeof value.id !== 'string' || typeof value.name !== 'string') {
      return null;
    }
    const id = value.id.trim();
    const name = value.name.trim();
    if (!id || !name) {
      return null;
    }
    return { id, name };
  } catch {
    return null;
  }
}
