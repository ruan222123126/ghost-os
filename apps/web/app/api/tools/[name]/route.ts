import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const toolPath = ({ name }: { name: string }): string => `/api/tools/${encodeURIComponent(name)}`;

export const PATCH = createParamBridgeRouteHandler('PATCH', toolPath);
