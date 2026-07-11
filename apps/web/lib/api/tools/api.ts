import type { ToolPayload, ToolUpdateRequest } from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { parseToolPayload, parseToolPayloadList } from '@/lib/api/tools/parser';

export async function listTools(): Promise<ToolPayload[]> {
  return requestJSON('/api/tools', {}, parseToolPayloadList);
}

export async function updateTool(name: string, input: ToolUpdateRequest): Promise<ToolPayload> {
  return requestJSON(`/api/tools/${encodeURIComponent(name)}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  }, parseToolPayload);
}
