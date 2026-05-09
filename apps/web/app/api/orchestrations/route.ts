import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const GET = createBridgeRouteHandler('GET', '/api/orchestrations');

export const POST = createBridgeRouteHandler('POST', '/api/orchestrations');
