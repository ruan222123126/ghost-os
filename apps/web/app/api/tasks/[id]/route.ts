import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const taskPath = ({ id }: { id: string }): string => `/api/tasks/${encodeURIComponent(id)}`;

export const GET = createParamBridgeRouteHandler('GET', taskPath);

export const PATCH = createParamBridgeRouteHandler('PATCH', taskPath);

export const DELETE = createParamBridgeRouteHandler('DELETE', taskPath);
