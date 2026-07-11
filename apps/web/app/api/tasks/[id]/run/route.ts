import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const taskRunPath = ({ id }: { id: string }): string => `/api/tasks/${encodeURIComponent(id)}/run`;

export const POST = createParamBridgeRouteHandler('POST', taskRunPath);
