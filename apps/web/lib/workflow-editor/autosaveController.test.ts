import {
  createAutosaveController,
  type AutosaveSnapshot,
  type AutosaveState,
} from '@/lib/workflow-editor/autosaveController';

const DEBOUNCE_MS = 800;
const RETRY_MS = 2000;

describe('workflow autosave controller', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('debounces frequent updates and persists latest snapshot', async () => {
    const persist = jest.fn<Promise<void>, [AutosaveSnapshot<string>]>().mockResolvedValue(undefined);
    const states: AutosaveState[] = [];
    const controller = createAutosaveController({
      debounceMs: DEBOUNCE_MS,
      retryMs: RETRY_MS,
      persist,
      onStateChange: (state) => states.push(state),
    });

    controller.schedule(buildSnapshot('A'));
    await advanceBy(400);
    controller.schedule(buildSnapshot('B'));
    await advanceBy(799);
    expect(persist).not.toHaveBeenCalled();

    await advanceBy(1);
    await flushAsync();
    expect(persist).toHaveBeenCalledTimes(1);
    expect(persist).toHaveBeenCalledWith(buildSnapshot('B'));
    expect(states.some((state) => state.phase === 'saving')).toBe(true);
    expect(states.some((state) => state.phase === 'saved')).toBe(true);

    controller.dispose();
  });

  it('flush saves immediately without waiting debounce', async () => {
    const persist = jest.fn<Promise<void>, [AutosaveSnapshot<string>]>().mockResolvedValue(undefined);
    const controller = createAutosaveController({
      debounceMs: DEBOUNCE_MS,
      retryMs: RETRY_MS,
      persist,
      onStateChange: () => undefined,
    });

    await controller.flush(buildSnapshot('instant'));
    expect(persist).toHaveBeenCalledTimes(1);
    expect(persist).toHaveBeenCalledWith(buildSnapshot('instant'));

    controller.dispose();
  });

  it('retries pending snapshot after failure', async () => {
    const states: AutosaveState[] = [];
    const persist = jest
      .fn<Promise<void>, [AutosaveSnapshot<string>]>()
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce(undefined);
    const controller = createAutosaveController({
      debounceMs: DEBOUNCE_MS,
      retryMs: RETRY_MS,
      persist,
      onStateChange: (state) => states.push(state),
    });

    controller.schedule(buildSnapshot('retry'));
    await advanceBy(DEBOUNCE_MS);
    await flushAsync();
    expect(persist).toHaveBeenCalledTimes(1);
    expect(states.some((state) => state.phase === 'error')).toBe(true);

    await advanceBy(RETRY_MS);
    await flushAsync();
    expect(persist).toHaveBeenCalledTimes(2);
    expect(persist.mock.calls[1][0]).toEqual(buildSnapshot('retry'));
    expect(states.some((state) => state.phase === 'saved')).toBe(true);

    controller.dispose();
  });

  it('does not persist snapshots already marked saved', async () => {
    const persist = jest.fn<Promise<void>, [AutosaveSnapshot<string>]>().mockResolvedValue(undefined);
    const controller = createAutosaveController({
      debounceMs: DEBOUNCE_MS,
      retryMs: RETRY_MS,
      persist,
      onStateChange: () => undefined,
    });

    const snapshot = buildSnapshot('same');
    controller.markSaved(snapshot.fingerprint);
    controller.schedule(snapshot);
    await advanceBy(DEBOUNCE_MS);
    await flushAsync();

    expect(persist).not.toHaveBeenCalled();
    controller.dispose();
  });

  it('flush with force persists snapshot even when marked saved', async () => {
    const persist = jest.fn<Promise<void>, [AutosaveSnapshot<string>]>().mockResolvedValue(undefined);
    const controller = createAutosaveController({
      debounceMs: DEBOUNCE_MS,
      retryMs: RETRY_MS,
      persist,
      onStateChange: () => undefined,
    });

    const snapshot = buildSnapshot('force');
    controller.markSaved(snapshot.fingerprint);
    await controller.flush(snapshot, { force: true });

    expect(persist).toHaveBeenCalledTimes(1);
    expect(persist).toHaveBeenCalledWith(snapshot);
    controller.dispose();
  });

  it('flush waits for in-flight persist before resolving', async () => {
    const firstPersist = createDeferred<void>();
    const secondPersist = createDeferred<void>();
    const persist = jest.fn<Promise<void>, [AutosaveSnapshot<string>]>().mockImplementation((snapshot) => {
      if (snapshot.fingerprint === 'first') {
        return firstPersist.promise;
      }
      if (snapshot.fingerprint === 'second') {
        return secondPersist.promise;
      }
      return Promise.resolve();
    });
    const controller = createAutosaveController({
      debounceMs: DEBOUNCE_MS,
      retryMs: RETRY_MS,
      persist,
      onStateChange: () => undefined,
    });

    controller.schedule(buildSnapshot('first'));
    await advanceBy(DEBOUNCE_MS);
    expect(persist).toHaveBeenCalledTimes(1);

    const flushPromise = controller.flush(buildSnapshot('second'), { force: true });
    await flushAsync();
    expect(persist).toHaveBeenCalledTimes(1);

    firstPersist.resolve();
    await flushAsync();
    expect(persist).toHaveBeenCalledTimes(2);
    expect(persist.mock.calls[1][0]).toEqual(buildSnapshot('second'));

    let settled = false;
    void flushPromise.then(() => {
      settled = true;
    });
    await flushAsync();
    expect(settled).toBe(false);

    secondPersist.resolve();
    await expect(flushPromise).resolves.toBeUndefined();
    controller.dispose();
  });
});

function buildSnapshot(payload: string): AutosaveSnapshot<string> {
  return {
    payload,
    fingerprint: payload,
  };
}

async function advanceBy(milliseconds: number): Promise<void> {
  jest.advanceTimersByTime(milliseconds);
  await flushAsync();
}

async function flushAsync(): Promise<void> {
  await Promise.resolve();
  await Promise.resolve();
}

function createDeferred<T>() {
  let resolve: (value: T | PromiseLike<T>) => void = () => undefined;
  let reject: (reason?: unknown) => void = () => undefined;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}
