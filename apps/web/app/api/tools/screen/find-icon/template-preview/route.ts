import { forwardBridgeDownload } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export async function GET(request: Request): Promise<Response> {
  const suffix = request.url.includes('?') ? request.url.slice(request.url.indexOf('?')) : '';
  return forwardBridgeDownload(`/api/tools/screen/find-icon/template${suffix}`, request);
}
