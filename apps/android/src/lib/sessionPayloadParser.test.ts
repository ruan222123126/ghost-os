import { describe, expect, it } from "vitest";
import { parseSessionDetail, parseSessionMetadataList } from "./sessionPayloadParser";

describe("session payload parser", () => {
  it("parses valid session metadata list", () => {
    const sessions = parseSessionMetadataList([sessionMetadata("session-1")]);

    expect(sessions).toEqual([sessionMetadata("session-1")]);
  });

  it("parses valid session detail", () => {
    const detail = parseSessionDetail({
      ...sessionMetadata("session-1"),
      messages: [
        {
          index: 0,
          role: "user",
          text: "hello",
        },
        {
          content: [{ type: "text", text: "done" }],
          index: 1,
          role: "assistant",
          thinking: "checking",
        },
      ],
      page: {
        before: null,
        has_more_before: false,
        limit: 100,
        next_before: null,
      },
    });

    expect(detail.id).toBe("session-1");
    expect(detail.messages).toHaveLength(2);
    expect(detail.page.limit).toBe(100);
  });

  it("rejects metadata entries missing required fields", () => {
    expect(() => parseSessionMetadataList([{ id: "session-1" }])).toThrow(
      "SESSIONS_LIST payload[0].title must be a string",
    );
  });

  it("rejects messages with invalid roles", () => {
    expect(() =>
      parseSessionDetail({
        ...sessionMetadata("session-1"),
        messages: [{ index: 0, role: "owner", text: "bad" }],
        page: { has_more_before: false, limit: 100 },
      }),
    ).toThrow("SESSION_GET payload.messages[0].role must be one of");
  });

  it("rejects detail payloads missing page fields", () => {
    expect(() =>
      parseSessionDetail({
        ...sessionMetadata("session-1"),
        messages: [],
        page: { limit: 100 },
      }),
    ).toThrow("SESSION_GET payload.page.has_more_before must be a boolean");
  });
});

function sessionMetadata(id: string) {
  return {
    created_at: "2026-01-01T00:00:00.000Z",
    id,
    message_count: 1,
    title: "Session",
    token_count: 2,
    updated_at: "2026-01-02T00:00:00.000Z",
  };
}
