import { describe, expect, it } from "vitest";

import type { DueItem } from "../../../api/payments";
import { monthName, transferTitle, zbpPayload } from "./transfer";

const month = (period: string): DueItem => ({ charge: period, kind: "contract_month", period, amount_minor: 20000, due_on: `${period}-05`, overdue: false });

describe("transferTitle", () => {
  it("names the learner and the months", () => {
    expect(transferTitle("Jan Kowalski", [month("2026-10")])).toBe("Lekcje Jan Kowalski 10.2026");
  });
  it("fits the ZBP title limit", () => {
    expect(transferTitle("Janina Wiśniewska-Kowalczyk", [month("2026-10"), month("2026-11")]).length).toBeLessThanOrEqual(32);
  });
});

describe("zbpPayload", () => {
  it("pads the amount in grosze and drops the country code from the account", () => {
    expect(zbpPayload("PL61109010140000071219812874", "Dominika Nowak", 20000, "Lekcje Jan 10.2026")).toBe("|PL|61109010140000071219812874|020000|Dominika Nowak|Lekcje Jan 10.2026|||");
  });
  it("clips the recipient and removes separators", () => {
    expect(zbpPayload("PL61109010140000071219812874", "Szkoła|Muzyczna Dominika Nowak", 100, "a|b")).toBe("|PL|61109010140000071219812874|000100|Szkoła Muzyczna Domi|a b|||");
  });
  it("returns nothing for an amount the code cannot hold", () => {
    expect(zbpPayload("PL61109010140000071219812874", "Dominika", 0, "x")).toBeNull();
    expect(zbpPayload("PL61109010140000071219812874", "Dominika", 1_000_000, "x")).toBeNull();
  });
});

describe("monthName", () => {
  it("uses the nominative month name", () => {
    expect(monthName("2026-10")).toBe("październik 2026");
  });
});
