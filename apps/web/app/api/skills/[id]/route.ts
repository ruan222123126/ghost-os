import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const skillPath = ({ id }: { id: string }): string => `/api/skills/${encodeURIComponent(id)}`;

export const DELETE = createParamBridgeRouteHandler('DELETE', skillPath);
