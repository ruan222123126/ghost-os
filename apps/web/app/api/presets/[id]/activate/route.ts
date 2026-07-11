import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const presetActivatePath = ({ id }: { id: string }): string => `/api/presets/${encodeURIComponent(id)}/activate`;

export const PUT = createParamBridgeRouteHandler('PUT', presetActivatePath);
