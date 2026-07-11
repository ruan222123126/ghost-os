import { requestJSON } from '@/lib/api/client';
import {
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  parseOptionalBoolean,
  pickKnownKeys,
  parseOptionalNumber,
} from '@/lib/api/shared';

const FIND_ICON_TEMPLATE_UPLOAD_KEYS = ['template_path', 'template_name', 'sha256'] as const;
const FIND_ICON_PREVIEW_KEYS = ['exists', 'match_count', 'matches', 'display_id', 'region', 'hovered'] as const;
const FIND_ICON_PREVIEW_REGION_KEYS = ['x', 'y', 'width', 'height'] as const;
const FIND_ICON_TEMPLATE_ENDPOINT = '/api/tools/screen/find-icon/template';

export interface FindIconTemplateUploadRequest {
  filename: string;
  mime_type: string;
  data_url: string;
  trace_id?: string;
}

export interface FindIconTemplateUploadResponse {
  template_path: string;
  template_name: string;
  sha256: string;
}

export interface FindIconPreviewRegion {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface FindIconPreviewRequest {
  template_path: string;
  threshold?: number;
  max_results?: number;
  display_id?: number;
  region?: FindIconPreviewRegion;
  hover_after_match?: boolean;
  trace_id?: string;
}

export interface FindIconPreviewResponse {
  exists: boolean;
  match_count: number;
  matches: Record<string, unknown>[];
  display_id?: number;
  region?: FindIconPreviewRegion;
  hovered?: boolean;
}

export async function uploadFindIconTemplate(input: FindIconTemplateUploadRequest): Promise<FindIconTemplateUploadResponse> {
  return requestJSON(FIND_ICON_TEMPLATE_ENDPOINT, {
    method: 'POST',
    body: JSON.stringify(input),
  }, parseFindIconTemplateUploadResponse);
}

export async function previewFindIcon(input: FindIconPreviewRequest): Promise<FindIconPreviewResponse> {
  return requestJSON('/api/tools/screen/find-icon/preview', {
    method: 'POST',
    body: JSON.stringify(input),
  }, parseFindIconPreviewResponse);
}

function parseFindIconTemplateUploadResponse(payload: unknown): FindIconTemplateUploadResponse {
  const record = pickKnownKeys(
    expectRecord(payload, 'find_icon template upload payload'),
    FIND_ICON_TEMPLATE_UPLOAD_KEYS,
  );
  return {
    template_path: expectString(record.template_path, 'find_icon template upload payload.template_path'),
    template_name: expectString(record.template_name, 'find_icon template upload payload.template_name'),
    sha256: expectString(record.sha256, 'find_icon template upload payload.sha256'),
  };
}

function parseFindIconPreviewResponse(payload: unknown): FindIconPreviewResponse {
  const record = pickKnownKeys(expectRecord(payload, 'find_icon preview payload'), FIND_ICON_PREVIEW_KEYS);
  const matches = parseFindIconMatches(record.matches);
  const region = parseOptionalFindIconRegion(record.region);

  return {
    exists: expectBoolean(record.exists, 'find_icon preview payload.exists'),
    match_count: expectNumber(record.match_count, 'find_icon preview payload.match_count'),
    matches,
    display_id: parseOptionalNumber(record.display_id, 'find_icon preview payload.display_id'),
    region,
    hovered: parseOptionalBoolean(record.hovered, 'find_icon preview payload.hovered'),
  };
}

function parseFindIconMatches(value: unknown): Record<string, unknown>[] {
  if (!Array.isArray(value)) {
    throw new Error('Invalid find_icon preview payload.matches: expected array');
  }
  return value.map((item, index) => expectRecord(item, `find_icon preview payload.matches[${index}]`));
}

function parseOptionalFindIconRegion(value: unknown): FindIconPreviewRegion | undefined {
  if (value === undefined) {
    return undefined;
  }
  const record = pickKnownKeys(expectRecord(value, 'find_icon preview payload.region'), FIND_ICON_PREVIEW_REGION_KEYS);
  return {
    x: expectNumber(record.x, 'find_icon preview payload.region.x'),
    y: expectNumber(record.y, 'find_icon preview payload.region.y'),
    width: expectNumber(record.width, 'find_icon preview payload.region.width'),
    height: expectNumber(record.height, 'find_icon preview payload.region.height'),
  };
}
