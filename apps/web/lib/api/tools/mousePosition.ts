import { requestJSON } from '@/lib/api/client';
import {
  expectNumber,
  expectRecord,
  parseOptionalNumber,
  pickKnownKeys,
} from '@/lib/api/shared';

const MOUSE_POSITION_KEYS = ['x', 'y', 'display_id', 'scale_x', 'scale_y'] as const;

export interface MousePositionRequest {
  trace_id?: string;
}

export interface MousePositionResponse {
  x: number;
  y: number;
  display_id?: number;
  scale_x?: number;
  scale_y?: number;
}

export async function getMousePosition(
  input: MousePositionRequest = {},
): Promise<MousePositionResponse> {
  return requestJSON(
    '/api/tools/screen/mouse-position',
    {
      method: 'POST',
      body: JSON.stringify(input),
    },
    parseMousePositionResponse,
  );
}

function parseMousePositionResponse(payload: unknown): MousePositionResponse {
  const record = pickKnownKeys(
    expectRecord(payload, 'mouse position payload'),
    MOUSE_POSITION_KEYS,
  );
  return {
    x: expectNumber(record.x, 'mouse position payload.x'),
    y: expectNumber(record.y, 'mouse position payload.y'),
    display_id: parseOptionalNumber(record.display_id, 'mouse position payload.display_id'),
    scale_x: parseOptionalNumber(record.scale_x, 'mouse position payload.scale_x'),
    scale_y: parseOptionalNumber(record.scale_y, 'mouse position payload.scale_y'),
  };
}
