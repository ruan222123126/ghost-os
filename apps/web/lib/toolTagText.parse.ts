import { isValidToolTagID } from './toolTagText.shared';
import type {
  ParsedToolTagCall,
  ToolTagParseResult,
  ToolTagParseState,
} from './toolTagText.types';

export function parseToolTagText(text: string): ToolTagParseResult {
  const state: ToolTagParseState = {
    mode: 'normal',
    rawTagStart: -1,
    toolIdBuffer: '',
    argsStart: -1,
    inString: false,
    escaped: false,
  };

  const visibleParts: string[] = [];
  const calls: ParsedToolTagCall[] = [];
  let cursor = 0;

  while (cursor < text.length) {
    if (state.mode === 'normal') {
      const openIndex = text.indexOf('<t:', cursor);
      if (openIndex < 0) {
        visibleParts.push(text.slice(cursor));
        cursor = text.length;
        break;
      }
      visibleParts.push(text.slice(cursor, openIndex));
      state.mode = 'capture_id';
      state.rawTagStart = openIndex;
      state.toolIdBuffer = '';
      cursor = openIndex + 3;
      continue;
    }

    if (state.mode === 'capture_id') {
      const closeOfID = text.indexOf('>', cursor);
      if (closeOfID < 0) {
        visibleParts.push(text.slice(state.rawTagStart));
        cursor = text.length;
        break;
      }
      const rawID = text.slice(cursor, closeOfID).trim();
      if (!isValidToolTagID(rawID)) {
        visibleParts.push(text.slice(state.rawTagStart, closeOfID + 1));
        state.mode = 'normal';
        cursor = closeOfID + 1;
        continue;
      }

      state.mode = 'capture_args';
      state.toolIdBuffer = rawID;
      state.argsStart = closeOfID + 1;
      state.inString = false;
      state.escaped = false;
      cursor = closeOfID + 1;
      continue;
    }

    const closeIndex = findToolTagCloseIndex(text, cursor, state);
    if (closeIndex < 0) {
      calls.push({
        toolId: state.toolIdBuffer,
        argsText: text.slice(state.argsStart),
      });
      cursor = text.length;
      break;
    }

    calls.push({
      toolId: state.toolIdBuffer,
      argsText: text.slice(state.argsStart, closeIndex),
    });
    state.mode = 'normal';
    cursor = closeIndex + 4;
  }

  return {
    visibleText: visibleParts.join(''),
    hasToolTags: calls.length > 0,
    calls,
  };
}

export function stripToolTagCalls(text: string): string {
  return parseToolTagText(text).visibleText;
}

export function hasToolTagCalls(text: string): boolean {
  return parseToolTagText(text).hasToolTags;
}

export function isPureToolTagDocument(text: string): boolean {
  const parsed = parseToolTagText(text);
  return parsed.hasToolTags && parsed.visibleText.trim().length === 0;
}

function findToolTagCloseIndex(text: string, cursor: number, state: ToolTagParseState): number {
  let index = cursor;
  while (index < text.length) {
    const char = text[index];
    if (state.inString) {
      if (state.escaped) {
        state.escaped = false;
        index += 1;
        continue;
      }
      if (char === '\\') {
        state.escaped = true;
        index += 1;
        continue;
      }
      if (char === '"') {
        state.inString = false;
      }
      index += 1;
      continue;
    }

    if (char === '"') {
      state.inString = true;
      index += 1;
      continue;
    }

    if (text.startsWith('</t>', index)) {
      return index;
    }
    index += 1;
  }
  return -1;
}
