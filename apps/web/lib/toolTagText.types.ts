export interface ParsedToolTagCall {
  toolId: string;
  argsText: string;
}

export interface ToolTagParseResult {
  visibleText: string;
  hasToolTags: boolean;
  calls: ParsedToolTagCall[];
}

export type ToolTagMode = 'normal' | 'capture_id' | 'capture_args';

export interface ToolTagParseState {
  mode: ToolTagMode;
  rawTagStart: number;
  toolIdBuffer: string;
  argsStart: number;
  inString: boolean;
  escaped: boolean;
}

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

export interface ToolTagStreamEventOpen {
  type: 'tool_open';
  callSeq: number;
  toolId: string;
}

export interface ToolTagStreamEventArgs {
  type: 'tool_args';
  callSeq: number;
  argsDelta: string;
}

export interface ToolTagStreamEventClose {
  type: 'tool_close';
  callSeq: number;
  toolId: string;
  argsText: string;
}

export type ToolTagStreamEvent = ToolTagStreamEventOpen | ToolTagStreamEventArgs | ToolTagStreamEventClose;

export interface ToolTagStreamUnitText {
  type: 'text';
  text: string;
}

export type ToolTagStreamUnit = ToolTagStreamUnitText | ToolTagStreamEvent;

export interface ToolTagStreamConsumeResult {
  visibleText: string;
  events: ToolTagStreamEvent[];
  units: ToolTagStreamUnit[];
}
