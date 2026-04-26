import type { SettingsCopy } from '@/lib/i18n/messages/settings';

export interface PromptSection {
  id: string;
  title: string;
  heading: string;
  content: string;
}

export function parsePromptSections(corePrompt: string, settings: SettingsCopy): PromptSection[] {
  const normalized = normalizePromptText(corePrompt);
  if (normalized === '') {
    return [newSection('section-0', settings.promptsSectionLabel(1), '', '')];
  }

  const headingSections = parseHeadingSections(normalized, settings);
  if (headingSections.length > 0) {
    return headingSections;
  }
  return parseParagraphSections(normalized, settings);
}

export function buildCorePromptFromSections(sections: readonly PromptSection[]): string {
  const blocks: string[] = [];
  for (const section of sections) {
    const body = section.content.trim();
    if (section.heading !== '') {
      blocks.push(body === '' ? `## ${section.heading}` : `## ${section.heading}\n${body}`);
      continue;
    }
    if (body !== '') {
      blocks.push(body);
    }
  }
  return blocks.join('\n\n').trim();
}

function parseHeadingSections(prompt: string, settings: SettingsCopy): PromptSection[] {
  const matches = Array.from(prompt.matchAll(/^##\s+(.+)$/gm));
  if (matches.length === 0) {
    return [];
  }

  const sections: PromptSection[] = [];
  const firstStart = matches[0].index ?? 0;
  if (firstStart > 0) {
    const preamble = prompt.slice(0, firstStart).trim();
    if (preamble !== '') {
      sections.push(newSection('preamble', settings.promptsPreambleLabel, '', preamble));
    }
  }

  for (let index = 0; index < matches.length; index += 1) {
    const start = matches[index].index ?? 0;
    const end = matches[index + 1]?.index ?? prompt.length;
    const block = prompt.slice(start, end).trim();
    const fallbackIndex = sections.length + 1;
    sections.push(sectionFromHeadingBlock(block, index, settings, fallbackIndex));
  }
  return sections;
}

function sectionFromHeadingBlock(
  block: string,
  index: number,
  settings: SettingsCopy,
  fallbackIndex: number,
): PromptSection {
  const lines = block.split('\n');
  const heading = lines[0].replace(/^##\s+/, '').trim();
  const title = heading === '' ? settings.promptsSectionLabel(fallbackIndex) : heading;
  const content = lines.slice(1).join('\n').trim();
  return newSection(`heading-${index}`, title, heading, content);
}

function parseParagraphSections(prompt: string, settings: SettingsCopy): PromptSection[] {
  const parts = prompt
    .split(/\n{2,}/)
    .map((part) => part.trim())
    .filter((part) => part !== '');

  if (parts.length === 0) {
    return [newSection('section-0', settings.promptsSectionLabel(1), '', '')];
  }

  return parts.map((part, index) => newSection(`section-${index}`, settings.promptsSectionLabel(index + 1), '', part));
}

function normalizePromptText(value: string): string {
  return value.replace(/\r\n/g, '\n').trim();
}

function newSection(id: string, title: string, heading: string, content: string): PromptSection {
  return { id, title, heading, content };
}
