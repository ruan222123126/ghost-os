import { bridgeUnavailableResponse, parseJSONBody, passThroughToBridge } from '@/lib/bridgeProxy';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
  const parsed = await parseJSONBody(request);
  if (!parsed.ok) {
    return parsed.response;
  }

  try {
    return await passThroughToBridge('/api/agent', {
      method: 'POST',
      body: JSON.stringify(parsed.body),
    });
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}
