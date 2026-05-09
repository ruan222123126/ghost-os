export const DEFAULT_INTERVAL_SECONDS = '300';
export const MIN_INTERVAL_SECONDS = 1;
export const DECIMAL_RADIX = 10;
export const START_NODE_TYPE = 'start';
export const END_NODE_TYPE = 'end';
export const DEFAULT_START_NODE_ID = 'start-node';
export const DEFAULT_END_NODE_ID = 'end-node';
export const AGENT_NODE_ID_PREFIX = 'agent-node';
export const NODE_X_GAP = 260;
export const NODE_Y_BASE = 80;
export const INPUT_NAME_MAX_LENGTH = 100;
export const INPUT_NAME_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/;
export const LOOP_ROLE_START = 'start';
export const LOOP_ROLE_END = 'end';
export const DEFAULT_LOOP_MAX_ITERATIONS = 3;
export const DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS = 3;
export const DEFAULT_ORCHESTRATION_GROUP_TITLE_PREFIX = '群组';
export const DEFAULT_ORCHESTRATION_AGENT_TITLE_PREFIX = '角色';
export const DEFAULT_ORCHESTRATION_SPEAKING_MODE = 'sequential';
export const WORKFLOW_IF_OPERATORS = [
  'equals',
  'not_equals',
  'contains',
  'not_contains',
  'is_empty',
  'not_empty',
] as const;
