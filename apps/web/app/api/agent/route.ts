// Next.js API route that starts agent runs and streams execution responses.

import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const POST = createBridgeRouteHandler('POST', '/api/agent');
