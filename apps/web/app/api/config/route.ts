// Next.js API route for reading and updating bridge configuration.

import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const GET = createBridgeRouteHandler('GET', '/api/config');

export const POST = createBridgeRouteHandler('POST', '/api/config');
