const GRAPHQL_TOOL_RESULT_MARKER = '[GRAPHQL_TOOL_RESULT]';
const GRAPHQL_CODE_FENCE_PATTERN = /```(?:graphql|gql)?\s*([\s\S]*?)```/i;

function extractGraphQLFenceContent(text: string): string | undefined {
  const match = GRAPHQL_CODE_FENCE_PATTERN.exec(text);
  return match?.[1]?.trim();
}

function stripGraphQLToolResultSuffix(text: string): string {
  const markerIndex = text.indexOf(GRAPHQL_TOOL_RESULT_MARKER);
  return markerIndex < 0 ? text : text.slice(0, markerIndex).trim();
}

function looksLikeGraphQLDocument(text: string): boolean {
  const trimmed = text.trim().toLowerCase();
  return trimmed.startsWith('{') || trimmed.startsWith('query') || trimmed.startsWith('mutation');
}

function hasBalancedGraphQLBraces(text: string): boolean {
  let depth = 0;
  let inString = false;
  let escaped = false;

  for (const character of text) {
    if (inString) {
      if (escaped) {
        escaped = false;
        continue;
      }
      if (character === '\\') {
        escaped = true;
        continue;
      }
      if (character === '"') {
        inString = false;
      }
      continue;
    }

    if (character === '"') {
      inString = true;
      continue;
    }
    if (character === '{') {
      depth += 1;
      continue;
    }
    if (character === '}') {
      depth -= 1;
      if (depth < 0) {
        return false;
      }
    }
  }

  return !inString && depth === 0;
}

export function normalizeGraphQLToolText(text: string): string {
  const trimmed = text.trim();
  const fenced = extractGraphQLFenceContent(trimmed);
  return stripGraphQLToolResultSuffix(fenced ?? trimmed).trim();
}

export function isGraphQLToolDocument(text: string): boolean {
  const normalized = normalizeGraphQLToolText(text);
  return normalized.length > 0 &&
    looksLikeGraphQLDocument(normalized) &&
    normalized.includes('{') &&
    normalized.endsWith('}') &&
    hasBalancedGraphQLBraces(normalized);
}
