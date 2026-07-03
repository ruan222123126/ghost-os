const SELECTED_SKILL_HEADER = "[Ghost-OS selected skill]";
const USER_REQUEST_HEADER = "[User request]";

export interface SelectedSkillIdentity {
  id: string;
  name: string;
}

export interface SelectedSkillMessageInput<TSkill extends SelectedSkillIdentity> {
  message: string;
  selectedSkill?: TSkill;
}

export interface ParsedSelectedSkillMessage<TSkill extends SelectedSkillIdentity> {
  message: string;
  selectedSkill?: TSkill;
}

export function buildSelectedSkillMessage<TSkill extends SelectedSkillIdentity>(
  input: SelectedSkillMessageInput<TSkill>,
): string {
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
    "The selected skill has already been loaded through sfind by the UI. Treat it as the active skill for this turn, do not call sfind just to load or verify it, and use the loaded skill for the user request. If the runtime does not provide this skill context, say that explicitly instead of pretending the skill is available. If the user request is empty, ask what task should be done with this skill.",
    "",
    USER_REQUEST_HEADER,
    message,
  ].join("\n");
}

export function parseSelectedSkillMessage<TSkill extends SelectedSkillIdentity>(
  raw: string,
): ParsedSelectedSkillMessage<TSkill> {
  if (!raw.startsWith(`${SELECTED_SKILL_HEADER}\n`)) {
    return { message: raw };
  }

  const marker = `\n${USER_REQUEST_HEADER}\n`;
  const markerIndex = raw.indexOf(marker);
  if (markerIndex < 0) {
    return { message: raw };
  }

  const metadata = parseSelectedSkillMetadata<TSkill>(
    raw.slice(SELECTED_SKILL_HEADER.length, markerIndex).trimStart(),
  );
  if (!metadata) {
    return { message: raw };
  }

  return {
    message: raw.slice(markerIndex + marker.length).trim(),
    selectedSkill: metadata,
  };
}

function parseSelectedSkillMetadata<TSkill extends SelectedSkillIdentity>(raw: string): TSkill | null {
  const firstLineEnd = raw.indexOf("\n");
  const metadataLine = (firstLineEnd >= 0 ? raw.slice(0, firstLineEnd) : raw).trim();
  if (!metadataLine) {
    return null;
  }

  try {
    const value = JSON.parse(metadataLine) as Record<string, unknown>;
    if (typeof value.id !== "string" || typeof value.name !== "string") {
      return null;
    }
    const id = value.id.trim();
    const name = value.name.trim();
    if (!id || !name) {
      return null;
    }
    return { id, name } as TSkill;
  } catch {
    return null;
  }
}
