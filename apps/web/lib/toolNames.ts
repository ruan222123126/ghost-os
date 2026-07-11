const TOOL_PREFIX = 'tools.';
export const SCRIPT_EXEC_TOOL = 'script_exec';

const PROMOTED_ACTION_TOOL_NAMES = new Set([
  'apply_diff',
  'bash_exec',
  'codex_cli',
  'codex_exec',
  'codex_patch',
  'codex_mcp',
  'codex_approval',
  'fetch_webpage',
  'list_files',
  'read_file',
  'screen_action',
  'screen_control',
  'search_files',
  'sfind',
  'web_search',
  'write_file',
]);

const TOOL_ALIASES: Record<string, string> = {
  apply_diff: 'apply_diff',
  bash: 'bash_exec',
  bash_exec: 'bash_exec',
  codex: 'codex_cli',
  codex_cli: 'codex_cli',
  codex_exec: 'codex_exec',
  codex_patch: 'codex_patch',
  codex_mcp: 'codex_mcp',
  codex_approval: 'codex_approval',
  edit: 'apply_diff',
  fetch_web: 'fetch_webpage',
  fetch_webpage: 'fetch_webpage',
  patch: 'apply_diff',
  read: 'read_file',
  read_file: 'read_file',
  run: 'bash_exec',
  screen_action: 'screen_action',
  screen_control: 'screen_control',
  script: SCRIPT_EXEC_TOOL,
  script_exec: SCRIPT_EXEC_TOOL,
  search: 'search_files',
  search_files: 'search_files',
  web: 'fetch_webpage',
  web_search: 'web_search',
  write: 'write_file',
  write_file: 'write_file',
};

export function normalizeToolName(rawName?: string): string {
  const normalized = rawName?.trim().toLowerCase() || '';
  if (!normalized) return '';
  const withoutPrefix = normalized.startsWith(TOOL_PREFIX)
    ? normalized.slice(TOOL_PREFIX.length)
    : normalized;
  if (TOOL_ALIASES[withoutPrefix]) return TOOL_ALIASES[withoutPrefix];
  for (const suffix of ['_exec', '_file']) {
    if (!withoutPrefix.endsWith(suffix)) continue;
    const baseName = withoutPrefix.slice(0, withoutPrefix.length - suffix.length);
    if (TOOL_ALIASES[baseName]) return TOOL_ALIASES[baseName];
  }
  return withoutPrefix;
}

export function supportsPromotedActionTitle(toolName?: string): boolean {
  return PROMOTED_ACTION_TOOL_NAMES.has(normalizeToolName(toolName));
}
