import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const GET = createBridgeRouteHandler('GET', '/api/config/providers');

export const POST = createBridgeRouteHandler('POST', '/api/config/providers');
