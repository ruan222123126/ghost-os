export {
  isPureToolTagDocument,
  parseToolTagText,
  stripToolTagCalls,
} from './toolTagText.parse';

export {
  consumeToolTagStreamChunk,
  createToolTagStreamState,
} from './toolTagText.stream';

export type {
  ParsedToolTagCall,
  ToolTagMode,
  ToolTagParseResult,
  ToolTagParseState,
  ToolTagStreamConsumeResult,
  ToolTagStreamEvent,
  ToolTagStreamEventArgs,
  ToolTagStreamEventClose,
  ToolTagStreamEventOpen,
  ToolTagStreamState,
  ToolTagStreamUnit,
  ToolTagStreamUnitText,
} from './toolTagText.types';
