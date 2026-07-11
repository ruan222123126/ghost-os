import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const GET = createBridgeRouteHandler('GET', '/api/tasks');

export const POST = createBridgeRouteHandler('POST', '/api/tasks');
