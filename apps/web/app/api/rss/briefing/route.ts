import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const GET = createBridgeRouteHandler('GET', '/api/rss/briefing');

export const POST = createBridgeRouteHandler('POST', '/api/rss/briefing');
