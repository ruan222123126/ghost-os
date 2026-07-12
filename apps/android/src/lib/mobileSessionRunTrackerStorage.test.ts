// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from "vitest";
import {
  loadMobileSessionRunTrackerState,
  saveMobileSessionRunTrackerState,
} from "./mobileSessionRunTrackerStorage";

describe("mobileSessionRunTrackerStorage", () => {
  beforeEach(() => window.localStorage.clear());

  it("persists tracker state per connection scope", () => {
    saveMobileSessionRunTrackerState("bridge-a", {
      connectionEpoch: 3,
      sessions: {
        "session-1": {
          sessionId: "session-1",
          status: "running",
          title: "Session 1",
          traceId: "trace-1",
          updatedAt: "2026-07-11T08:00:00Z",
        },
      },
    });

    expect(loadMobileSessionRunTrackerState("bridge-a").sessions["session-1"]?.traceId).toBe("trace-1");
    expect(loadMobileSessionRunTrackerState("bridge-b").sessions).toEqual({});
  });

  it("surfaces damaged storage and resets it", () => {
    window.localStorage.setItem("ghost-os-mobile.sessionRunTracker.v1:bridge-a", "{");

    expect(loadMobileSessionRunTrackerState("bridge-a")).toEqual({ connectionEpoch: 0, sessions: {} });
  });
});
