import { createParamBridgeRouteHandler } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

const providerPath = ({ name }: { name: string }): string => `/api/config/providers/${encodeURIComponent(name)}`;

export const PUT = createParamBridgeRouteHandler('PUT', providerPath);

export const DELETE = createParamBridgeRouteHandler('DELETE', providerPath);
