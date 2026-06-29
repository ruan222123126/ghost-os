// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  cancelAllStreamReplyCommitters,
  cancelStreamReplyCommit,
  createStreamReplyCommitter,
  enqueueStreamReplyCommit,
  flushStreamReplyCommit,
} from "./streamReplyCommitter";

describe("streamReplyCommitter", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("coalesces multiple enqueues into one scheduled commit", () => {
    let scheduled: FrameRequestCallback | undefined;
    const rafSpy = vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      scheduled = callback;
      return 1;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => {});
    const commit = vi.fn();
    const committer = createStreamReplyCommitter();

    committer.enqueue(commit);
    committer.enqueue(commit);

    expect(rafSpy).toHaveBeenCalledTimes(1);
    expect(commit).toHaveBeenCalledTimes(0);
    scheduled?.(0);
    expect(commit).toHaveBeenCalledTimes(1);
  });

  it("flushes the latest pending commit immediately", () => {
    let scheduled: FrameRequestCallback | undefined;
    const rafSpy = vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      scheduled = callback;
      return 1;
    });
    const cancelSpy = vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => {});
    const commit = vi.fn();
    const committer = createStreamReplyCommitter();

    committer.enqueue(commit);
    committer.flush();

    expect(rafSpy).toHaveBeenCalledTimes(1);
    expect(cancelSpy).toHaveBeenCalledTimes(1);
    expect(commit).toHaveBeenCalledTimes(1);
    scheduled?.(0);
    expect(commit).toHaveBeenCalledTimes(1);
  });

  it("cancels all pending committers without invoking callbacks", () => {
    const committers = new Map<string, ReturnType<typeof createStreamReplyCommitter>>();
    const commit = vi.fn();

    enqueueStreamReplyCommit(committers, "request-1", commit);
    cancelStreamReplyCommit(committers, "request-1");
    cancelAllStreamReplyCommitters(committers);

    expect(commit).not.toHaveBeenCalled();
    expect(committers.size).toBe(0);
  });

  it("flushStreamReplyCommit removes the committer after commit", () => {
    const committers = new Map<string, ReturnType<typeof createStreamReplyCommitter>>();
    const commit = vi.fn();

    enqueueStreamReplyCommit(committers, "request-1", commit);
    flushStreamReplyCommit(committers, "request-1");

    expect(commit).toHaveBeenCalledTimes(1);
    expect(committers.size).toBe(0);
  });
});
