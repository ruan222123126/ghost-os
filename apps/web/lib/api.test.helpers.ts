export const fetchMock = jest.fn();

export function installFetchMock() {
  fetchMock.mockReset();
  Object.defineProperty(global, 'fetch', {
    value: fetchMock,
    writable: true,
  });
}

export function mockFetchJSON(
  body: unknown,
  options: { ok?: boolean; status?: number } = {}
) {
  const raw = JSON.stringify(body);
  fetchMock.mockResolvedValue({
    ok: options.ok ?? true,
    status: options.status ?? 200,
    headers: new Headers({ 'content-type': 'application/json' }),
    json: async () => body,
    text: async () => raw,
  });
}
