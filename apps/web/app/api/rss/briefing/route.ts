import { createBridgeRouteHandler } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

export const GET = createBridgeRouteHandler('GET', '/api/rss/briefing');

export const POST = createBridgeRouteHandler('POST', '/api/rss/briefing');
