import type { RecordValue } from './records';

interface InlineArgSpec {
  targetKey: string;
  sourceName: string;
  fallbackFirstLiteral?: boolean;
}

const INLINE_ARG_SPECS = buildInlineArgSpecs();

export function parseInlineArgs(toolName: string, argsText: string): RecordValue {
  const firstLiteral = readFirstQuotedLiteral(argsText);
  const args: RecordValue = {};
  for (const spec of INLINE_ARG_SPECS[toolName] ?? []) {
    const value = readInlineArgValue(argsText, firstLiteral, spec);
    if (value) {
      args[spec.targetKey] = value;
    }
  }
  return args;
}

function readInlineArgValue(argsText: string, firstLiteral: string, spec: InlineArgSpec): string {
  return readNamedQuotedArg(argsText, spec.sourceName) || (spec.fallbackFirstLiteral ? firstLiteral : '');
}

function buildInlineArgSpecs(): Record<string, InlineArgSpec[]> {
  const pathSpec = { targetKey: 'path', sourceName: 'path', fallbackFirstLiteral: true };
  return {
    list_files: [pathSpec],
    read_file: [pathSpec],
    write_file: [pathSpec, { targetKey: 'content', sourceName: 'content' }],
    apply_diff: [pathSpec],
    bash_exec: [{ targetKey: 'command', sourceName: 'command', fallbackFirstLiteral: true }],
    fetch_webpage: [{ targetKey: 'url', sourceName: 'url', fallbackFirstLiteral: true }],
    web_search: [{ targetKey: 'query', sourceName: 'query', fallbackFirstLiteral: true }],
    search_files: [{ targetKey: 'query', sourceName: 'query', fallbackFirstLiteral: true }],
  };
}

function readNamedQuotedArg(argsText: string, name: string): string {
  const pattern = new RegExp(`${name}\\s*=\\s*(['"])(.*?)\\1`, 's');
  const match = argsText.match(pattern);
  return match?.[2]?.trim() || '';
}

function readFirstQuotedLiteral(argsText: string): string {
  const match = argsText.match(/^\s*(['"])(.*?)\1/s);
  return match?.[2]?.trim() || '';
}
