import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const sessionEventsPath = ({ id }: { id: string }): string => {
  return `/api/sessions/${encodeURIComponent(id)}/events`;
};

export const GET = createParamBridgeRouteHandler('GET', sessionEventsPath);
