import { createBridgeRouteHandler } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

export const PUT = createBridgeRouteHandler('PUT', '/api/config/active-provider');
