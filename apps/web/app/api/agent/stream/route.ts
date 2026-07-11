// Next.js API route that starts agent runs and proxies bridge SSE events.

import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const POST = createBridgeRouteHandler('POST', '/api/agent/stream');
