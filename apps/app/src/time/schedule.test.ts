import { describe, expect, it } from "vitest";

import { futureExceptionDraft, groupSlotsByLocalDate, localInputToUtc, utcToLocalInput } from "./schedule";

describe("schedule display boundaries", () => {
  it("round trips a concrete instant through a teacher-local input", () => {
    const input = utcToLocalInput("2026-01-15T12:00:00Z", "America/New_York");
    expect(input).toBe("2026-01-15T07:00");
    expect(localInputToUtc(input, "America/New_York")).toBe("2026-01-15T12:00:00.000Z");
  });

  it("groups horizon slots by the viewer's local calendar date", () => {
    const groups = groupSlotsByLocalDate([
      { start_at: "2026-01-15T23:30:00Z" },
      { start_at: "2026-01-16T00:15:00Z" },
    ], "America/New_York");
    expect([...groups.values()].map((items) => items.length)).toEqual([2]);
    expect([...groups.keys()]).toEqual(["2026-01-15"]);
  });

  it("rejects impossible and ambiguous named-zone DST inputs", () => {
    expect(() => localInputToUtc("2026-03-08T02:30", "America/New_York")).toThrow("nie istnieje");
    expect(() => localInputToUtc("2026-11-01T01:30", "America/New_York")).toThrow("niejednoznaczna");
  });

  it("creates an aligned future exception interval with a 45-minute duration", () => {
    const draft = futureExceptionDraft(new Date("2026-01-15T12:07:00Z"));
    const start = localInputToUtc(draft.start);
    const end = localInputToUtc(draft.end);
    expect(new Date(end).getTime() - new Date(start).getTime()).toBe(45 * 60000);
    expect(draft.start).toMatch(/T\d\d:(00|15|30|45)$/);
  });
});
