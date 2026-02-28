import { bridgeUnavailableResponse, passThroughToBridge } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

interface RouteContext {
  params: {
    id: string;
  };
}

export async function GET(_request: Request, context: RouteContext) {
  try {
    return await passThroughToBridge(`/api/sessions/${encodeURIComponent(context.params.id)}`, {
      method: 'GET',
    });
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}

export async function DELETE(_request: Request, context: RouteContext) {
  try {
    return await passThroughToBridge(`/api/sessions/${encodeURIComponent(context.params.id)}`, {
      method: 'DELETE',
    });
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}
