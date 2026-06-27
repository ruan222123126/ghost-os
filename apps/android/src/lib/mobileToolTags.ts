const OPEN_TOKEN = "<t:";
const CLOSE_TOKEN = "</t>";
const NO_ACTIVE_TOOL_CALL = 0;

type ToolTagMode = "normal" | "capture_id" | "capture_args";

export interface ToolTagStreamState {
  mode: ToolTagMode;
  normalCandidate: string;
  rawTagPrefix: string;
  currentToolId: string;
  closeCandidate: string;
  argsBuffer: string;
  inString: boolean;
  escaped: boolean;
  currentCallSeq: number;
  nextCallSeq: number;
}

export interface ParsedToolTagCall {
  toolId: string;
  argsText: string;
}

export interface ToolTagParseResult {
  visibleText: string;
  calls: ParsedToolTagCall[];
  units: ParsedToolTagParseUnit[];
}

export interface ParsedToolTagTextUnit {
  type: "text";
  text: string;
}

export interface ParsedToolTagCallUnit {
  type: "tool_call";
  toolId: string;
  argsText: string;
}

export type ParsedToolTagParseUnit = ParsedToolTagTextUnit | ParsedToolTagCallUnit;

export interface ToolTagStreamEventOpen {
  type: "tool_open";
  callSeq: number;
  toolId: string;
}

export interface ToolTagStreamEventArgs {
  type: "tool_args";
  callSeq: number;
  argsDelta: string;
}

export interface ToolTagStreamEventClose {
  type: "tool_close";
  callSeq: number;
  toolId: string;
  argsText: string;
}

export type ToolTagStreamEvent = ToolTagStreamEventOpen | ToolTagStreamEventArgs | ToolTagStreamEventClose;

export interface ToolTagStreamUnitText {
  type: "text";
  text: string;
}

export type ToolTagStreamUnit = ToolTagStreamEvent | ToolTagStreamUnitText;

export interface ToolTagStreamConsumeResult {
  visibleText: string;
  events: ToolTagStreamEvent[];
  units: ToolTagStreamUnit[];
}

interface ToolTagParseState {
  mode: ToolTagMode;
  rawTagStart: number;
  toolIdBuffer: string;
  argsStart: number;
  inString: boolean;
  escaped: boolean;
}

export function createToolTagStreamState(): ToolTagStreamState {
  return {
    argsBuffer: "",
    closeCandidate: "",
    currentToolId: "",
    escaped: false,
    inString: false,
    mode: "normal",
    normalCandidate: "",
    currentCallSeq: 0,
    nextCallSeq: 1,
    rawTagPrefix: "",
  };
}

export function consumeToolTagStreamChunk(
  state: ToolTagStreamState,
  chunk: string,
  finalize = false,
): ToolTagStreamConsumeResult {
  if (!chunk && !finalize) {
    return { events: [], units: [], visibleText: "" };
  }

  const textParts: string[] = [];
  const events: ToolTagStreamEvent[] = [];
  const units: ToolTagStreamUnit[] = [];
  for (const char of chunk) {
    if (state.mode === "normal") {
      consumeNormalChar(state, char, textParts, units);
      continue;
    }
    if (state.mode === "capture_id") {
      consumeCaptureIdChar(state, char, textParts, events, units);
      continue;
    }
    consumeCaptureArgsChar(state, char, events, units);
  }

  if (finalize) {
    finalizeToolTagStreamState(state, textParts, events, units);
  }

  return { events, units, visibleText: textParts.join("") };
}

export function parseToolTagText(text: string): ToolTagParseResult {
  const state: ToolTagParseState = {
    argsStart: -1,
    escaped: false,
    inString: false,
    mode: "normal",
    rawTagStart: -1,
    toolIdBuffer: "",
  };

  const visibleParts: string[] = [];
  const calls: ParsedToolTagCall[] = [];
  const units: ParsedToolTagParseUnit[] = [];
  let cursor = 0;
  while (cursor < text.length) {
    if (state.mode === "normal") {
      const openIndex = text.indexOf(OPEN_TOKEN, cursor);
      if (openIndex < 0) {
        pushParsedTextUnit(visibleParts, units, text.slice(cursor));
        break;
      }
      pushParsedTextUnit(visibleParts, units, text.slice(cursor, openIndex));
      state.mode = "capture_id";
      state.rawTagStart = openIndex;
      state.toolIdBuffer = "";
      cursor = openIndex + OPEN_TOKEN.length;
      continue;
    }

    if (state.mode === "capture_id") {
      const idEnd = text.indexOf(">", cursor);
      if (idEnd < 0) {
        visibleParts.push(text.slice(state.rawTagStart));
        break;
      }
      const rawId = text.slice(cursor, idEnd).trim();
      if (!isValidToolTagId(rawId)) {
        pushParsedTextUnit(visibleParts, units, text.slice(state.rawTagStart, idEnd + 1));
        state.mode = "normal";
        cursor = idEnd + 1;
        continue;
      }

      state.mode = "capture_args";
      state.toolIdBuffer = rawId;
      state.argsStart = idEnd + 1;
      state.inString = false;
      state.escaped = false;
      cursor = idEnd + 1;
      continue;
    }

    const closeIndex = findToolTagCloseIndex(text, cursor, state);
    if (closeIndex < 0) {
      const call = {
        argsText: text.slice(state.argsStart),
        toolId: state.toolIdBuffer,
      } satisfies ParsedToolTagCall;
      calls.push(call);
      units.push({ ...call, type: "tool_call" });
      break;
    }
    const call = {
      argsText: text.slice(state.argsStart, closeIndex),
      toolId: state.toolIdBuffer,
    } satisfies ParsedToolTagCall;
    calls.push(call);
    units.push({ ...call, type: "tool_call" });
    state.mode = "normal";
    cursor = closeIndex + CLOSE_TOKEN.length;
  }

  return { calls, units, visibleText: visibleParts.join("") };
}

export function stripToolTagCalls(text: string): string {
  return parseToolTagText(text).visibleText;
}

function consumeNormalChar(
  state: ToolTagStreamState,
  char: string,
  textParts: string[],
  units: ToolTagStreamUnit[],
): void {
  state.normalCandidate += char;
  while (state.normalCandidate.length > 0) {
    if (OPEN_TOKEN.startsWith(state.normalCandidate)) {
      if (state.normalCandidate === OPEN_TOKEN) {
        state.mode = "capture_id";
        state.rawTagPrefix = OPEN_TOKEN;
        state.currentToolId = "";
        state.normalCandidate = "";
      }
      return;
    }
    pushVisibleText(textParts, units, state.normalCandidate.slice(0, 1));
    state.normalCandidate = state.normalCandidate.slice(1);
  }
}

function consumeCaptureIdChar(
  state: ToolTagStreamState,
  char: string,
  textParts: string[],
  events: ToolTagStreamEvent[],
  units: ToolTagStreamUnit[],
): void {
  state.rawTagPrefix += char;
  if (char !== ">") {
    state.currentToolId += char;
    return;
  }

  const toolId = state.currentToolId.trim();
  if (!isValidToolTagId(toolId)) {
    pushVisibleText(textParts, units, state.rawTagPrefix);
    resetToNormal(state);
    return;
  }

  state.mode = "capture_args";
  state.currentToolId = toolId;
  state.argsBuffer = "";
  state.closeCandidate = "";
  state.inString = false;
  state.escaped = false;
  state.currentCallSeq = state.nextCallSeq;
  state.nextCallSeq += 1;
  state.rawTagPrefix = "";
  recordToolTagEvent(events, units, { callSeq: state.currentCallSeq, toolId, type: "tool_open" });
}

function consumeCaptureArgsChar(
  state: ToolTagStreamState,
  char: string,
  events: ToolTagStreamEvent[],
  units: ToolTagStreamUnit[],
): void {
  if (!state.inString) {
    state.closeCandidate += char;
    if (CLOSE_TOKEN.startsWith(state.closeCandidate)) {
      if (state.closeCandidate === CLOSE_TOKEN) {
        recordToolTagEvent(events, units, {
          argsText: state.argsBuffer,
          callSeq: state.currentCallSeq,
          toolId: state.currentToolId,
          type: "tool_close",
        });
        resetToNormal(state);
      }
      return;
    }
    flushCloseCandidateAsArgs(state, events, units);
    return;
  }
  appendArgsChar(state, char, events, units);
}

function flushCloseCandidateAsArgs(
  state: ToolTagStreamState,
  events: ToolTagStreamEvent[],
  units: ToolTagStreamUnit[],
): void {
  while (state.closeCandidate.length > 0 && !CLOSE_TOKEN.startsWith(state.closeCandidate)) {
    const nextChar = state.closeCandidate.slice(0, 1);
    state.closeCandidate = state.closeCandidate.slice(1);
    appendArgsChar(state, nextChar, events, units);
  }
}

function appendArgsChar(
  state: ToolTagStreamState,
  char: string,
  events: ToolTagStreamEvent[],
  units: ToolTagStreamUnit[],
): void {
  state.argsBuffer += char;
  recordToolTagEvent(events, units, { argsDelta: char, callSeq: state.currentCallSeq, type: "tool_args" });

  if (state.inString) {
    if (state.escaped) {
      state.escaped = false;
      return;
    }
    if (char === "\\") {
      state.escaped = true;
      return;
    }
    if (char === "\"") {
      state.inString = false;
    }
    return;
  }

  if (char === "\"") {
    state.inString = true;
  }
}

function finalizeToolTagStreamState(
  state: ToolTagStreamState,
  textParts: string[],
  events: ToolTagStreamEvent[],
  units: ToolTagStreamUnit[],
): void {
  if (state.mode === "normal") {
    if (state.normalCandidate) {
      pushVisibleText(textParts, units, state.normalCandidate);
      state.normalCandidate = "";
    }
    return;
  }

  if (state.mode === "capture_id") {
    pushVisibleText(textParts, units, state.rawTagPrefix);
    resetToNormal(state);
    return;
  }

  if (state.closeCandidate) {
    for (const char of state.closeCandidate) {
      appendArgsChar(state, char, events, units);
    }
    state.closeCandidate = "";
  }
  recordToolTagEvent(events, units, {
    argsText: state.argsBuffer,
    callSeq: state.currentCallSeq,
    toolId: state.currentToolId,
    type: "tool_close",
  });
  resetToNormal(state);
}

function pushVisibleText(
  textParts: string[],
  units: ToolTagStreamUnit[],
  text: string,
): void {
  if (!text) {
    return;
  }
  textParts.push(text);
  const lastUnit = units[units.length - 1];
  if (lastUnit?.type === "text") {
    lastUnit.text = `${lastUnit.text}${text}`;
    return;
  }
  units.push({ text, type: "text" });
}

function recordToolTagEvent(
  events: ToolTagStreamEvent[],
  units: ToolTagStreamUnit[],
  event: ToolTagStreamEvent,
): void {
  events.push(event);
  units.push(event);
}

function pushParsedTextUnit(
  visibleParts: string[],
  units: ParsedToolTagParseUnit[],
  text: string,
): void {
  if (!text) {
    return;
  }
  visibleParts.push(text);
  const lastUnit = units[units.length - 1];
  if (lastUnit?.type === "text") {
    lastUnit.text = `${lastUnit.text}${text}`;
    return;
  }
  units.push({ text, type: "text" });
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
      if (char === "\\") {
        state.escaped = true;
        index += 1;
        continue;
      }
      if (char === "\"") {
        state.inString = false;
      }
      index += 1;
      continue;
    }

    if (char === "\"") {
      state.inString = true;
      index += 1;
      continue;
    }

    if (text.startsWith(CLOSE_TOKEN, index)) {
      return index;
    }
    index += 1;
  }
  return -1;
}

function resetToNormal(state: ToolTagStreamState): void {
  state.mode = "normal";
  state.normalCandidate = "";
  state.rawTagPrefix = "";
  state.currentToolId = "";
  state.closeCandidate = "";
  state.argsBuffer = "";
  state.inString = false;
  state.escaped = false;
  state.currentCallSeq = 0;
}

function isValidToolTagId(value: string): boolean {
  if (!value) {
    return false;
  }
  for (const char of value) {
    if (char < "0" || char > "9") {
      return false;
    }
  }
  return Number.parseInt(value, 10) > NO_ACTIVE_TOOL_CALL;
}
