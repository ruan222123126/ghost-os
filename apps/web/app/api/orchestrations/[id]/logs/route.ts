import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const GET = createParamBridgeRouteHandler(
  'GET',
  ({ id }: { id: string }) => `/api/orchestrations/${encodeURIComponent(id)}/logs`,
);
