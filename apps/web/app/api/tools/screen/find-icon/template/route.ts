import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const GET = createBridgeRouteHandler('GET', '/api/tools/screen/find-icon/template');
export const POST = createBridgeRouteHandler('POST', '/api/tools/screen/find-icon/template');
