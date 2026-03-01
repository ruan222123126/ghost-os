// Next.js API route for session collection operations (create/list).

import { bridgeUnavailableResponse, passThroughToBridge } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

export async function GET() {
  try {
    return await passThroughToBridge('/api/sessions', { method: 'GET' });
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}
