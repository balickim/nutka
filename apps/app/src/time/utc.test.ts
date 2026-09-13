import { describe, expect, it } from "vitest";

import { formatUtcInstant, isUtcInstant, localizeUtcInstant, parseUtcInstant } from "./utc";

describe("UTC scheduling helpers", () => {
  it("accepts only valid RFC3339 UTC instants", () => {
    expect(parseUtcInstant("2026-03-29T00:15:00Z").toISOString()).toBe("2026-03-29T00:15:00.000Z");
    expect(isUtcInstant("2026-03-29T00:15:00+02:00")).toBe(false);
    expect(isUtcInstant("2026-02-30T00:15:00Z")).toBe(false);
  });

  it("formats and localizes without changing the stored instant", () => {
    const instant = parseUtcInstant("2026-01-15T12:00:00Z");
    expect(formatUtcInstant(instant)).toBe("2026-01-15T12:00:00.000Z");
    expect(localizeUtcInstant("2026-01-15T12:00:00Z", "en-US", { timeZone: "America/New_York", hour: "numeric", minute: "numeric", timeZoneName: "short" })).toContain("7:00");
  });
});
