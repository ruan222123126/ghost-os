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
  fetchMock.mockResolvedValue({
    ok: options.ok ?? true,
    status: options.status ?? 200,
    json: async () => body,
  });
}
