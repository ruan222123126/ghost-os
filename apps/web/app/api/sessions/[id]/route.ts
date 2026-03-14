// Module-level helpers and contracts for this file.

import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const sessionPath = ({ id }: { id: string }): string => `/api/sessions/${encodeURIComponent(id)}`;

export const GET = createParamBridgeRouteHandler('GET', sessionPath);

export const DELETE = createParamBridgeRouteHandler('DELETE', sessionPath);
