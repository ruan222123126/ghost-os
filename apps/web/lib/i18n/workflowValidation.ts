import type { WebLocale } from '@/lib/i18n/locale';

const QUOTED_VALUE_PATTERN = /"([^"]*)"/g;
const NUMBER_PATTERN = /(\d+)/g;

const exactMatchers: Array<{ en: string; zh: string }> = [
  { en: 'workflow node id is required', zh: '工作流节点 id 不能为空' },
  { en: 'workflow edge endpoints are required', zh: '工作流边的起点和终点不能为空' },
  { en: 'workflow requires exactly 1 start node', zh: '工作流必须且仅能有 1 个 start 节点' },
  { en: 'workflow requires exactly 1 end node', zh: '工作流必须且仅能有 1 个 end 节点' },
  { en: 'workflow start/end nodes must both be present or both be absent', zh: '工作流的 start/end 节点必须同时存在或同时不存在' },
  { en: 'workflow requires at least 1 group node', zh: '编排至少需要 1 个群组节点' },
  { en: 'workflow requires exactly 1 entry group', zh: '编排必须且仅能有 1 个入口群组' },
  { en: 'workflow requires exactly 1 exit group', zh: '编排必须且仅能有 1 个出口群组' },
  { en: 'Validation failed, not saved', zh: '校验失败，未保存' },
  { en: 'failed to build workflow payload', zh: '构建工作流载荷失败' },
];

const regexMatchers: Array<{ pattern: RegExp; zh: (matches: RegExpMatchArray) => string }> = [
  {
    pattern: /^duplicate workflow node id "([^"]+)"$/,
    zh: (matches) => `重复的工作流节点 id："${matches[1]}"`,
  },
  {
    pattern: /^workflow does not allow self-loop edge "([^"]+)"$/,
    zh: (matches) => `不允许自环边："${matches[1]}"`,
  },
  {
    pattern: /^workflow edge references unknown from_node_id "([^"]+)"$/,
    zh: (matches) => `工作流边引用了未知 from_node_id："${matches[1]}"`,
  },
  {
    pattern: /^workflow edge references unknown to_node_id "([^"]+)"$/,
    zh: (matches) => `工作流边引用了未知 to_node_id："${matches[1]}"`,
  },
  {
    pattern: /^duplicate workflow edge "([^"]+)"$/,
    zh: (matches) => `重复的工作流边："${matches[1]}"`,
  },
  {
    pattern: /^workflow node "([^"]+)" payload does not match type "([^"]+)"$/,
    zh: (matches) => `节点 "${matches[1]}" 的 payload 与类型 "${matches[2]}" 不匹配`,
  },
  {
    pattern: /^workflow tool node "([^"]+)" requires tool_name$/,
    zh: (matches) => `工具节点 "${matches[1]}" 需要 tool_name`,
  },
  {
    pattern: /^workflow llm node "([^"]+)" requires prompt$/,
    zh: (matches) => `LLM 节点 "${matches[1]}" 需要 prompt`,
  },
  {
    pattern: /^workflow agent node "([^"]+)" requires message$/,
    zh: (matches) => `Agent 节点 "${matches[1]}" 需要 message`,
  },
  {
    pattern: /^workflow agent node "([^"]+)" preset_id "([^"]+)" is not available$/,
    zh: (matches) => `Agent 节点 "${matches[1]}" 的 preset_id "${matches[2]}" 不可用`,
  },
  {
    pattern: /^workflow agent node "([^"]+)" tool_allowlist contains unavailable tool "([^"]+)"$/,
    zh: (matches) => `Agent 节点 "${matches[1]}" 的 tool_allowlist 包含不可用工具 "${matches[2]}"`,
  },
  {
    pattern: /^workflow if node "([^"]+)" requires true_node_id and false_node_id$/,
    zh: (matches) => `If 节点 "${matches[1]}" 需要 true_node_id 和 false_node_id`,
  },
  {
    pattern: /^workflow if node "([^"]+)" true_node_id and false_node_id must differ$/,
    zh: (matches) => `If 节点 "${matches[1]}" 的 true_node_id 与 false_node_id 不能相同`,
  },
  {
    pattern: /^workflow loop node "([^"]+)" requires loop_id$/,
    zh: (matches) => `Loop 节点 "${matches[1]}" 需要 loop_id`,
  },
  {
    pattern: /^workflow loop node "([^"]+)" requires role=start\|end$/,
    zh: (matches) => `Loop 节点 "${matches[1]}" 的 role 必须为 start|end`,
  },
  {
    pattern: /^workflow loop node "([^"]+)" start role requires max_iterations > 0$/,
    zh: (matches) => `Loop 节点 "${matches[1]}" 的 start 角色需要 max_iterations > 0`,
  },
  {
    pattern: /^unsupported workflow node type "([^"]+)"$/,
    zh: (matches) => `不支持的节点类型："${matches[1]}"`,
  },
  {
    pattern: /^workflow group node "([^"]+)" must have in<=1 and out<=1$/,
    zh: (matches) => `群组节点 "${matches[1]}" 的控制流入度和出度都不能超过 1`,
  },
  {
    pattern: /^orchestration group node "([^"]+)" requires owner_agent_id in owner mode$/,
    zh: (matches) => `编排群组节点 "${matches[1]}" 在群主模式下必须设置 owner_agent_id`,
  },
  {
    pattern: /^orchestration group node "([^"]+)" owner_agent_id "([^"]+)" must be an existing member$/,
    zh: (matches) => `编排群组节点 "${matches[1]}" 的 owner_agent_id "${matches[2]}" 必须属于当前成员`,
  },
  {
    pattern: /^orchestration group node "([^"]+)" requires speaking_mode sequential\|parallel\|owner$/,
    zh: (matches) => `编排群组节点 "${matches[1]}" 的 speaking_mode 必须是 sequential|parallel|owner`,
  },
];

export function localizeWorkflowValidationError(message: string, locale: WebLocale): string {
  const trimmed = message.trim();
  if (!trimmed) {
    return trimmed;
  }
  if (locale === 'en-US') {
    return trimmed;
  }

  for (const matcher of exactMatchers) {
    if (trimmed === matcher.en) {
      return matcher.zh;
    }
  }

  for (const matcher of regexMatchers) {
    const matches = trimmed.match(matcher.pattern);
    if (matches) {
      return matcher.zh(matches);
    }
  }

  return fallbackKeywordLocalization(trimmed);
}

function fallbackKeywordLocalization(message: string): string {
  const quoted = message.match(QUOTED_VALUE_PATTERN) ?? [];
  const numbers = message.match(NUMBER_PATTERN) ?? [];

  let localized = message
    .replace('workflow', '工作流')
    .replace('node', '节点')
    .replace('edge', '边')
    .replace('requires', '需要')
    .replace('duplicate', '重复')
    .replace('unknown', '未知')
    .replace('payload', '载荷')
    .replace('type', '类型')
    .replace('must be', '必须是')
    .replace('length must be <=', '长度必须 <=')
    .replace('name must be a non-empty string', '名称不能为空')
    .replace('name must match', '名称必须匹配');

  quoted.forEach((value) => {
    localized = localized.replace(value, value);
  });
  numbers.forEach((value) => {
    localized = localized.replace(value, value);
  });

  return localized;
}
