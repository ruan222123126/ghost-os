import { isValidToolTagID } from './toolTagText.shared';
import type {
  ParsedToolTagCall,
  ToolTagParseResult,
  ToolTagParseState,
} from './toolTagText.types';

interface ToolTagParseContext {
  calls: ParsedToolTagCall[];
  cursor: number;
  state: ToolTagParseState;
  text: string;
  visibleParts: string[];
}

export function parseToolTagText(text: string): ToolTagParseResult {
  const context = createToolTagParseContext(text);
  while (context.cursor < text.length) {
    parseNextToolTagSpan(context);
  }

  return {
    visibleText: context.visibleParts.join(''),
    hasToolTags: context.calls.length > 0,
    calls: context.calls,
  };
}

export function stripToolTagCalls(text: string): string {
  return parseToolTagText(text).visibleText;
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

function createToolTagParseContext(text: string): ToolTagParseContext {
  return {
    calls: [],
    cursor: 0,
    state: {
      mode: 'normal',
      rawTagStart: -1,
      toolIdBuffer: '',
      argsStart: -1,
      inString: false,
      escaped: false,
    },
    text,
    visibleParts: [],
  };
}

function parseNextToolTagSpan(context: ToolTagParseContext): void {
  if (context.state.mode === 'normal') {
    parseNormalSpan(context);
    return;
  }
  if (context.state.mode === 'capture_id') {
    parseToolIdSpan(context);
    return;
  }
  parseArgsSpan(context);
}

function parseNormalSpan(context: ToolTagParseContext): void {
  const openIndex = context.text.indexOf('<t:', context.cursor);
  if (openIndex < 0) {
    context.visibleParts.push(context.text.slice(context.cursor));
    context.cursor = context.text.length;
    return;
  }
  context.visibleParts.push(context.text.slice(context.cursor, openIndex));
  context.state.mode = 'capture_id';
  context.state.rawTagStart = openIndex;
  context.state.toolIdBuffer = '';
  context.cursor = openIndex + 3;
}

function parseToolIdSpan(context: ToolTagParseContext): void {
  const closeOfID = context.text.indexOf('>', context.cursor);
  if (closeOfID < 0) {
    context.visibleParts.push(context.text.slice(context.state.rawTagStart));
    context.cursor = context.text.length;
    return;
  }

  const rawID = context.text.slice(context.cursor, closeOfID).trim();
  if (!isValidToolTagID(rawID)) {
    context.visibleParts.push(context.text.slice(context.state.rawTagStart, closeOfID + 1));
    context.state.mode = 'normal';
    context.cursor = closeOfID + 1;
    return;
  }

  context.state.mode = 'capture_args';
  context.state.toolIdBuffer = rawID;
  context.state.argsStart = closeOfID + 1;
  context.state.inString = false;
  context.state.escaped = false;
  context.cursor = closeOfID + 1;
}

function parseArgsSpan(context: ToolTagParseContext): void {
  const closeIndex = findToolTagCloseIndex(context.text, context.cursor, context.state);
  if (closeIndex < 0) {
    context.calls.push({
      toolId: context.state.toolIdBuffer,
      argsText: context.text.slice(context.state.argsStart),
    });
    context.cursor = context.text.length;
    return;
  }

  context.calls.push({
    toolId: context.state.toolIdBuffer,
    argsText: context.text.slice(context.state.argsStart, closeIndex),
  });
  context.state.mode = 'normal';
  context.cursor = closeIndex + 4;
}
