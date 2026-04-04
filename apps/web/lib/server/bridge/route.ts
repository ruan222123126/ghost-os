import { forwardBridge } from './forward';
import type { BridgeMethod, BridgeRouteContext, BridgeRouteHandlerOptions } from './types';

export function createBridgeRouteHandler(
  method: BridgeMethod,
  path: string,
  options: BridgeRouteHandlerOptions<Record<string, never>> = {}
) {
  return createParamBridgeRouteHandler<Record<string, never>>(method, () => path, options);
}

export function createParamBridgeRouteHandler<Params extends Record<string, string>>(
  method: BridgeMethod,
  getPath: (params: Params) => string,
  options: BridgeRouteHandlerOptions<Params> = {}
) {
  return async function routeHandler(request: Request, context?: BridgeRouteContext<Params>): Promise<Response> {
    const params = (context?.params ?? {}) as Params;
    const headers = typeof options.headers === 'function' ? options.headers({ params, request }) : options.headers;
    return forwardBridge({
      path: appendRequestSearch(getPath(params), request),
      method,
      request,
      headers,
    });
  };
}

function appendRequestSearch(path: string, request: Request): string {
  const search = new URL(request.url).search;
  if (!search) {
    return path;
  }
  return `${path}${search}`;
}
