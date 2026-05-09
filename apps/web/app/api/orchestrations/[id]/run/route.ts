import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

export const POST = createParamBridgeRouteHandler(
  'POST',
  ({ id }: { id: string }) => `/api/orchestrations/${encodeURIComponent(id)}/run`,
);
