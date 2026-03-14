// Next.js API route that forwards generic bus envelopes to the bridge backend.

import { createBridgeRouteHandler } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

export const POST = createBridgeRouteHandler('POST', '/api/bus');
