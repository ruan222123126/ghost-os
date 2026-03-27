// Next.js API route that submits a human answer and proxies bridge SSE events.

import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const POST = createBridgeRouteHandler('POST', '/api/questions/answer/stream', {
  headers: ({ request }) => ({
    'X-Trace-ID': request.headers.get('X-Trace-ID')?.trim() || `web-${Date.now()}`,
  }),
});
