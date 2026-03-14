// Next.js API route that submits a human answer and lets the bridge resume the paused agent.

import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const POST = createBridgeRouteHandler('POST', '/api/questions/answer', {
  headers: () => ({
    'X-Trace-ID': `web-${Date.now()}`,
  }),
});
