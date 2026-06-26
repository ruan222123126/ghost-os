import { CLOSE_TOKEN, OPEN_TOKEN, isValidToolTagID } from './toolTagText.shared';
import type {
  ToolTagStreamConsumeResult,
  ToolTagStreamEvent,
  ToolTagStreamState,
  ToolTagStreamUnit,
} from './toolTagText.types';

interface ToolTagStreamOutput {
  events: ToolTagStreamEvent[];
  textParts: string[];
  units: ToolTagStreamUnit[];
}

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

  const output: ToolTagStreamOutput = {
    events: [],
    textParts: [],
    units: [],
  };
  for (const char of chunk) {
    if (state.mode === 'normal') {
      consumeNormalChar(state, char, output);
      continue;
    }
    if (state.mode === 'capture_id') {
      consumeCaptureIDChar(state, char, output);
      continue;
    }
    consumeCaptureArgsChar(state, char, output);
  }

  if (finalize) {
    finalizeToolTagStreamState(state, output);
  }

  return {
    visibleText: output.textParts.join(''),
    events: output.events,
    units: output.units,
  };
}

function consumeNormalChar(
  state: ToolTagStreamState,
  char: string,
  output: ToolTagStreamOutput,
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
    appendTextPart(output, state.normalCandidate.slice(0, 1));
    state.normalCandidate = state.normalCandidate.slice(1);
  }
}

function consumeCaptureIDChar(
  state: ToolTagStreamState,
  char: string,
  output: ToolTagStreamOutput,
): void {
  state.rawTagPrefix += char;
  if (char !== '>') {
    state.currentToolId += char;
    return;
  }

  const toolID = state.currentToolId.trim();
  if (!isValidToolTagID(toolID)) {
    appendTextPart(output, state.rawTagPrefix);
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
  appendToolStreamEvent(output, {
    type: 'tool_open',
    callSeq: state.currentCallSeq,
    toolId: state.currentToolId,
  });
}

function consumeCaptureArgsChar(
  state: ToolTagStreamState,
  char: string,
  output: ToolTagStreamOutput,
): void {
  if (!state.inString) {
    state.closeCandidate += char;
    if (CLOSE_TOKEN.startsWith(state.closeCandidate)) {
      if (state.closeCandidate === CLOSE_TOKEN) {
        appendToolStreamEvent(output, {
          type: 'tool_close',
          callSeq: state.currentCallSeq,
          toolId: state.currentToolId,
          argsText: state.argsBuffer,
        });
        resetToNormal(state);
      }
      return;
    }
    flushCloseCandidateAsArgs(state, output);
    return;
  }
  appendArgsChar(state, char, output);
}

function flushCloseCandidateAsArgs(
  state: ToolTagStreamState,
  output: ToolTagStreamOutput,
): void {
  while (state.closeCandidate.length > 0 && !CLOSE_TOKEN.startsWith(state.closeCandidate)) {
    const nextChar = state.closeCandidate.slice(0, 1);
    state.closeCandidate = state.closeCandidate.slice(1);
    appendArgsChar(state, nextChar, output);
  }
}

function appendArgsChar(
  state: ToolTagStreamState,
  char: string,
  output: ToolTagStreamOutput,
): void {
  state.argsBuffer += char;
  appendArgsDelta(output, state.currentCallSeq, char);

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
  output: ToolTagStreamOutput,
  callSeq: number,
  char: string,
): void {
  appendToolStreamEvent(output, {
    type: 'tool_args',
    callSeq,
    argsDelta: char,
  });
}

function finalizeToolTagStreamState(
  state: ToolTagStreamState,
  output: ToolTagStreamOutput,
): void {
  if (state.mode === 'normal') {
    if (state.normalCandidate) {
      appendTextPart(output, state.normalCandidate);
      state.normalCandidate = '';
    }
    return;
  }

  if (state.mode === 'capture_id') {
    appendTextPart(output, state.rawTagPrefix);
    resetToNormal(state);
    return;
  }

  if (state.closeCandidate) {
    for (const char of state.closeCandidate) {
      appendArgsChar(state, char, output);
    }
    state.closeCandidate = '';
  }
  appendToolStreamEvent(output, {
    type: 'tool_close',
    callSeq: state.currentCallSeq,
    toolId: state.currentToolId,
    argsText: state.argsBuffer,
  });
  resetToNormal(state);
}

function appendTextPart(
  output: ToolTagStreamOutput,
  text: string,
): void {
  if (!text) {
    return;
  }
  output.textParts.push(text);
  const last = output.units[output.units.length - 1];
  if (last && last.type === 'text') {
    last.text += text;
    return;
  }
  output.units.push({
    type: 'text',
    text,
  });
}

function appendToolStreamEvent(
  output: ToolTagStreamOutput,
  event: ToolTagStreamEvent,
): void {
  if (event.type !== 'tool_args') {
    output.events.push(event);
    output.units.push(event);
    return;
  }

  const lastEvent = output.events[output.events.length - 1];
  if (lastEvent && lastEvent.type === 'tool_args' && lastEvent.callSeq === event.callSeq) {
    lastEvent.argsDelta += event.argsDelta;
  } else {
    output.events.push({
      ...event,
    });
  }

  const lastUnit = output.units[output.units.length - 1];
  if (lastUnit && lastUnit.type === 'tool_args' && lastUnit.callSeq === event.callSeq) {
    lastUnit.argsDelta += event.argsDelta;
    return;
  }
  output.units.push({
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
