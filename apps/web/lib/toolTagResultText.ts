const TOOL_TAG_RESULT_PREFIX = '[TOOL_TAG_RESULT]';
const TOOL_SEARCH_NAME = 'sfind';
const LOADED_TOOL_STATUSES = new Set([
  'active',
  'already_loaded',
  'loaded',
  'pending',
]);

type RecordValue = Record<string, unknown>;

export function filterToolTagResultToLoadedTools(text: string): string {
  const payload = parseToolTagResultPayload(text);
  if (!payload || readString(payload, 'tool') !== TOOL_SEARCH_NAME) {
    return text;
  }

  const output = parsePayloadOutput(payload.output);
  const items = readItemList(output?.items);
  if (!output || !items) {
    return text;
  }

  const filteredItems = items.filter((item) => isLoadedToolItem(item));
  if (filteredItems.length === items.length) {
    return text;
  }

  return `${TOOL_TAG_RESULT_PREFIX}\n${JSON.stringify({
    ...payload,
    output: {
      ...output,
      items: filteredItems,
    },
  })}`;
}

function parseToolTagResultPayload(text: string): RecordValue | undefined {
  const markerIndex = text.indexOf(TOOL_TAG_RESULT_PREFIX);
  if (markerIndex < 0) {
    return undefined;
  }
  const payloadText = text.slice(markerIndex + TOOL_TAG_RESULT_PREFIX.length).trim();
  return parseRecordValue(payloadText);
}

function parsePayloadOutput(output: unknown): RecordValue | undefined {
  if (isRecordValue(output)) {
    return output;
  }
  if (typeof output !== 'string') {
    return undefined;
  }
  return parseRecordValue(output);
}

function parseRecordValue(raw: string): RecordValue | undefined {
  if (!raw) {
    return undefined;
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    return isRecordValue(parsed) ? parsed : undefined;
  } catch {
    return undefined;
  }
}

function readItemList(value: unknown): RecordValue[] | undefined {
  if (!Array.isArray(value)) {
    return undefined;
  }
  return value.filter((item): item is RecordValue => isRecordValue(item));
}

function isLoadedToolItem(item: RecordValue): boolean {
  const status = readString(item, 'status').toLowerCase();
  if (status && LOADED_TOOL_STATUSES.has(status)) {
    return true;
  }
  return readBoolean(item, 'available_now') || readBoolean(item, 'available_next_turn');
}

function readString(value: RecordValue, key: string): string {
  const raw = value[key];
  return typeof raw === 'string' ? raw.trim() : '';
}

function readBoolean(value: RecordValue, key: string): boolean {
  return value[key] === true;
}

function isRecordValue(value: unknown): value is RecordValue {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
