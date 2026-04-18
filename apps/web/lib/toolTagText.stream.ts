import { CLOSE_TOKEN, OPEN_TOKEN, isValidToolTagID } from './toolTagText.shared';
import type {
  ToolTagStreamConsumeResult,
  ToolTagStreamEvent,
  ToolTagStreamState,
  ToolTagStreamUnit,
} from './toolTagText.types';

export function createToolTagStreamState(): ToolTagStreamState {
  return {
    mode: 'normal',
    normalCandidate: '',
    rawTagPrefix: '',
    currentToolId: '',
    closeCandidate: '',
    argsBuffer: '',
    inString: false,
    escaped: false,
    currentCallSeq: 0,
    nextCallSeq: 1,
  };
}

export function consumeToolTagStreamChunk(
  state: ToolTagStreamState,
  chunk: string,
  finalize = false,
): ToolTagStreamConsumeResult {
  if (!chunk && !finalize) {
    return { visibleText: '', events: [], units: [] };
  }

  const textParts: string[] = [];
  const events: ToolTagStreamEvent[] = [];
  const units: ToolTagStreamUnit[] = [];
  for (const char of chunk) {
    if (state.mode === 'normal') {
      consumeNormalChar(state, char, textParts, units);
      continue;
    }
    if (state.mode === 'capture_id') {
      consumeCaptureIDChar(state, char, textParts, events, units);
      continue;
    }
    consumeCaptureArgsChar(state, char, events, units);
  }

  if (finalize) {
    finalizeToolTagStreamState(state, textParts, events, units);
  }

  return {
    visibleText: textParts.join(''),
    events,
    units,
  };
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
        state.mode = 'capture_id';
        state.rawTagPrefix = OPEN_TOKEN;
        state.currentToolId = '';
        state.normalCandidate = '';
      }
      return;
    }
    appendTextPart(textParts, units, state.normalCandidate.slice(0, 1));
    state.normalCandidate = state.normalCandidate.slice(1);
  }
}

function consumeCaptureIDChar(
  state: ToolTagStreamState,
  char: string,
  textParts: string[],
  events: ToolTagStreamEvent[],
  units: ToolTagStreamUnit[],
): void {
  state.rawTagPrefix += char;
  if (char !== '>') {
    state.currentToolId += char;
    return;
  }

  const toolID = state.currentToolId.trim();
  if (!isValidToolTagID(toolID)) {
    appendTextPart(textParts, units, state.rawTagPrefix);
    resetToNormal(state);
    return;
  }

  state.mode = 'capture_args';
  state.currentToolId = toolID;
  state.argsBuffer = '';
  state.closeCandidate = '';
  state.inString = false;
  state.escaped = false;
  state.currentCallSeq = state.nextCallSeq;
  state.nextCallSeq += 1;
  state.rawTagPrefix = '';
  appendToolStreamEvent(events, units, {
    type: 'tool_open',
    callSeq: state.currentCallSeq,
    toolId: state.currentToolId,
  });
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
        appendToolStreamEvent(events, units, {
          type: 'tool_close',
          callSeq: state.currentCallSeq,
          toolId: state.currentToolId,
          argsText: state.argsBuffer,
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
  appendArgsDelta(events, units, state.currentCallSeq, char);

  if (state.inString) {
    if (state.escaped) {
      state.escaped = false;
      return;
    }
    if (char === '\\') {
      state.escaped = true;
      return;
    }
    if (char === '"') {
      state.inString = false;
    }
    return;
  }

  if (char === '"') {
    state.inString = true;
  }
}

function appendArgsDelta(
  events: ToolTagStreamEvent[],
  units: ToolTagStreamUnit[],
  callSeq: number,
  char: string,
): void {
  appendToolStreamEvent(events, units, {
    type: 'tool_args',
    callSeq,
    argsDelta: char,
  });
}

function finalizeToolTagStreamState(
  state: ToolTagStreamState,
  textParts: string[],
  events: ToolTagStreamEvent[],
  units: ToolTagStreamUnit[],
): void {
  if (state.mode === 'normal') {
    if (state.normalCandidate) {
      appendTextPart(textParts, units, state.normalCandidate);
      state.normalCandidate = '';
    }
    return;
  }

  if (state.mode === 'capture_id') {
    appendTextPart(textParts, units, state.rawTagPrefix);
    resetToNormal(state);
    return;
  }

  if (state.closeCandidate) {
    for (const char of state.closeCandidate) {
      appendArgsChar(state, char, events, units);
    }
    state.closeCandidate = '';
  }
  appendToolStreamEvent(events, units, {
    type: 'tool_close',
    callSeq: state.currentCallSeq,
    toolId: state.currentToolId,
    argsText: state.argsBuffer,
  });
  resetToNormal(state);
}

function appendTextPart(
  textParts: string[],
  units: ToolTagStreamUnit[],
  text: string,
): void {
  if (!text) {
    return;
  }
  textParts.push(text);
  const last = units[units.length - 1];
  if (last && last.type === 'text') {
    last.text += text;
    return;
  }
  units.push({
    type: 'text',
    text,
  });
}

function appendToolStreamEvent(
  events: ToolTagStreamEvent[],
  units: ToolTagStreamUnit[],
  event: ToolTagStreamEvent,
): void {
  if (event.type !== 'tool_args') {
    events.push(event);
    units.push(event);
    return;
  }

  const lastEvent = events[events.length - 1];
  if (lastEvent && lastEvent.type === 'tool_args' && lastEvent.callSeq === event.callSeq) {
    lastEvent.argsDelta += event.argsDelta;
  } else {
    events.push({
      ...event,
    });
  }

  const lastUnit = units[units.length - 1];
  if (lastUnit && lastUnit.type === 'tool_args' && lastUnit.callSeq === event.callSeq) {
    lastUnit.argsDelta += event.argsDelta;
    return;
  }
  units.push({
    ...event,
  });
}

function resetToNormal(state: ToolTagStreamState): void {
  state.mode = 'normal';
  state.normalCandidate = '';
  state.rawTagPrefix = '';
  state.currentToolId = '';
  state.closeCandidate = '';
  state.argsBuffer = '';
  state.inString = false;
  state.escaped = false;
  state.currentCallSeq = 0;
}
