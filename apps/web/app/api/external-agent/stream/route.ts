// Next.js API route that proxies external Codex agent SSE events to the bridge.

import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const POST = createBridgeRouteHandler('POST', '/api/external-agent/stream', {
  headers: ({ request }) => {
    const traceId = request.headers.get('X-Trace-ID')?.trim();
    if (!traceId) {
      return undefined;
    }

    return {
      'X-Trace-ID': traceId,
    };
  },
});
