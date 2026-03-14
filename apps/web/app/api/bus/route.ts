// Next.js API route that forwards generic bus envelopes to the bridge backend.

import { createBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const POST = createBridgeRouteHandler('POST', '/api/bus');
