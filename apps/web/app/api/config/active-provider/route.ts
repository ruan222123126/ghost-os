import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const PUT = createBridgeRouteHandler('PUT', '/api/config/active-provider');
