import { forwardBridgeDownload } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

export async function GET(
  request: Request,
  context: { params: { id: string; artifactId: string } }
): Promise<Response> {
  const { id, artifactId } = context.params;
  const suffix = request.url.includes('?') ? request.url.slice(request.url.indexOf('?')) : '';
  return forwardBridgeDownload(
    `/api/sessions/${encodeURIComponent(id)}/artifacts/${encodeURIComponent(artifactId)}${suffix}`,
    request,
  );
}
