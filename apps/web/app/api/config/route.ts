// Next.js API route for reading and updating bridge configuration.

import { bridgeUnavailableResponse, parseJSONBody, passThroughToBridge } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

export async function GET() {
  try {
    return await passThroughToBridge('/api/config', { method: 'GET' });
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}

export async function POST(request: Request) {
  const parsed = await parseJSONBody(request);
  if (!parsed.ok) {
    return parsed.response;
  }

  try {
    return await passThroughToBridge('/api/config', {
      method: 'POST',
      body: JSON.stringify(parsed.body),
    });
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}
