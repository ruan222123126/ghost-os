export interface ForwardBridgeOptions {
  path: string;
  method: NonNullable<RequestInit['method']>;
  request?: Request;
  headers?: HeadersInit;
}

export type BridgeMethod = ForwardBridgeOptions['method'];

export interface BridgeRouteContext<Params extends Record<string, string>> {
  params: Params;
}

export type BridgeRouteHeaders<Params extends Record<string, string>> =
  | HeadersInit
  | ((context: { params: Params; request: Request }) => HeadersInit | undefined);

export interface BridgeRouteHandlerOptions<Params extends Record<string, string>> {
  headers?: BridgeRouteHeaders<Params>;
}
