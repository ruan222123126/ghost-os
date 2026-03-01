// Next.js API route that forwards generic bus envelopes to the bridge backend.

import { bridgeUnavailableResponse, parseJSONBody, passThroughToBridge } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
  const parsed = await parseJSONBody(request);
  if (!parsed.ok) {
    return parsed.response;
  }

  try {
    return await passThroughToBridge('/api/bus', {
      method: 'POST',
      body: JSON.stringify(parsed.body),
    });
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}
