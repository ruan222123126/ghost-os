import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const taskStopPath = ({ id }: { id: string }): string => `/api/tasks/${encodeURIComponent(id)}/stop`;

export const POST = createParamBridgeRouteHandler('POST', taskStopPath);
