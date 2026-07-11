import { createParamBridgeRouteHandler } from '@/lib/server/bridge';

export const dynamic = 'force-dynamic';

const pathForID = (id: string) => `/api/orchestrations/${encodeURIComponent(id)}`;

export const GET = createParamBridgeRouteHandler(
  'GET',
  ({ id }: { id: string }) => pathForID(id),
);

export const PATCH = createParamBridgeRouteHandler(
  'PATCH',
  ({ id }: { id: string }) => pathForID(id),
);

export const DELETE = createParamBridgeRouteHandler(
  'DELETE',
  ({ id }: { id: string }) => pathForID(id),
);
