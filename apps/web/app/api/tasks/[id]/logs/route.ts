import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const taskLogsPath = ({ id }: { id: string }): string => `/api/tasks/${encodeURIComponent(id)}/logs`;

export const GET = createParamBridgeRouteHandler('GET', taskLogsPath);
