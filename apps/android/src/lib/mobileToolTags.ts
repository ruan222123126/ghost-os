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
    rawTagPrefix: "",
  };
}

export function consumeToolTagStreamChunk(state: ToolTagStreamState, chunk: string, finalize = false): string {
  if (!chunk && !finalize) {
    return "";
  }

  const textParts: string[] = [];
  for (const char of chunk) {
    if (state.mode === "normal") {
      consumeNormalChar(state, char, textParts);
      continue;
    }
    if (state.mode === "capture_id") {
      consumeCaptureIdChar(state, char, textParts);
      continue;
    }
    consumeCaptureArgsChar(state, char);
  }

  if (finalize) {
    finalizeToolTagStreamState(state, textParts);
  }

  return textParts.join("");
}

export function stripToolTagCalls(text: string): string {
  const state: ToolTagParseState = {
    argsStart: -1,
    escaped: false,
    inString: false,
    mode: "normal",
    rawTagStart: -1,
    toolIdBuffer: "",
  };

  const visibleParts: string[] = [];
  let cursor = 0;
  while (cursor < text.length) {
    if (state.mode === "normal") {
      const openIndex = text.indexOf(OPEN_TOKEN, cursor);
      if (openIndex < 0) {
        visibleParts.push(text.slice(cursor));
        break;
      }
      visibleParts.push(text.slice(cursor, openIndex));
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
        visibleParts.push(text.slice(state.rawTagStart, idEnd + 1));
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
      break;
    }
    state.mode = "normal";
    cursor = closeIndex + CLOSE_TOKEN.length;
  }

  return visibleParts.join("");
}

function consumeNormalChar(state: ToolTagStreamState, char: string, textParts: string[]): void {
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
    textParts.push(state.normalCandidate.slice(0, 1));
    state.normalCandidate = state.normalCandidate.slice(1);
  }
}

function consumeCaptureIdChar(state: ToolTagStreamState, char: string, textParts: string[]): void {
  state.rawTagPrefix += char;
  if (char !== ">") {
    state.currentToolId += char;
    return;
  }

  const toolId = state.currentToolId.trim();
  if (!isValidToolTagId(toolId)) {
    textParts.push(state.rawTagPrefix);
    resetToNormal(state);
    return;
  }

  state.mode = "capture_args";
  state.currentToolId = toolId;
  state.argsBuffer = "";
  state.closeCandidate = "";
  state.inString = false;
  state.escaped = false;
  state.rawTagPrefix = "";
}

function consumeCaptureArgsChar(state: ToolTagStreamState, char: string): void {
  if (!state.inString) {
    state.closeCandidate += char;
    if (CLOSE_TOKEN.startsWith(state.closeCandidate)) {
      if (state.closeCandidate === CLOSE_TOKEN) {
        resetToNormal(state);
      }
      return;
    }
    flushCloseCandidateAsArgs(state);
    return;
  }
  appendArgsChar(state, char);
}

function flushCloseCandidateAsArgs(state: ToolTagStreamState): void {
  while (state.closeCandidate.length > 0 && !CLOSE_TOKEN.startsWith(state.closeCandidate)) {
    const nextChar = state.closeCandidate.slice(0, 1);
    state.closeCandidate = state.closeCandidate.slice(1);
    appendArgsChar(state, nextChar);
  }
}

function appendArgsChar(state: ToolTagStreamState, char: string): void {
  state.argsBuffer += char;

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

function finalizeToolTagStreamState(state: ToolTagStreamState, textParts: string[]): void {
  if (state.mode === "normal") {
    if (state.normalCandidate) {
      textParts.push(state.normalCandidate);
      state.normalCandidate = "";
    }
    return;
  }

  if (state.mode === "capture_id") {
    textParts.push(state.rawTagPrefix);
    resetToNormal(state);
    return;
  }

  resetToNormal(state);
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
