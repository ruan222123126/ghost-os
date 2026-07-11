// Next.js API route for reading and updating system prompt files.

import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const GET = createBridgeRouteHandler('GET', '/api/prompts/system');

export const PATCH = createBridgeRouteHandler('PATCH', '/api/prompts/system');
