// Next.js API route for session collection operations (create/list).

import { createBridgeRouteHandler } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

export const GET = createBridgeRouteHandler('GET', '/api/sessions');
