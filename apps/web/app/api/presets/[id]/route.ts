import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const presetPath = ({ id }: { id: string }): string => `/api/presets/${encodeURIComponent(id)}`;

export const PATCH = createParamBridgeRouteHandler('PATCH', presetPath);

export const DELETE = createParamBridgeRouteHandler('DELETE', presetPath);
