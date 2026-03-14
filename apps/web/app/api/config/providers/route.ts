import { createBridgeRouteHandler } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

export const GET = createBridgeRouteHandler('GET', '/api/config/providers');

export const POST = createBridgeRouteHandler('POST', '/api/config/providers');
