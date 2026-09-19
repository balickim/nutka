import { describe, expect, it } from "vitest";

import type { PracticeSummary } from "../../../api/practice";
import { addDays, dayChoices, learnerPracticeLine, practiceGrid, teacherPracticeLine } from "./days";

const summary = (days: number, minutes: number): PracticeSummary => ({ since_on: "2030-03-01", days, minutes, sessions: days, tasks: [], comments: [] });

describe("practice days", () => {
  it("moves by calendar days across daylight saving and month ends", () => {
    expect(addDays("2030-03-31", -1)).toBe("2030-03-30");
    expect(addDays("2030-03-01", -1)).toBe("2030-02-28");
  });

  it("offers today, yesterday, and dated days back to the backfill limit", () => {
    const choices = dayChoices("2030-03-02", 14);
    expect(choices).toHaveLength(15);
    expect(choices.slice(0, 2)).toEqual([{ value: "2030-03-02", label: "Dziś" }, { value: "2030-03-01", label: "Wczoraj" }]);
    expect(choices[14].value).toBe("2030-02-16");
    expect(choices[2].label).toContain("28 lutego");
  });

  it("builds four Monday weeks that end with the week of today", () => {
    const grid = practiceGrid("2030-03-06", ["2030-03-04", "2030-02-11"]);
    expect(grid).toHaveLength(4);
    expect(grid[0][0].date).toBe("2030-02-11");
    expect(grid[3][6].date).toBe("2030-03-10");
    expect(grid.flat().filter((day) => day.practiced).map((day) => day.date)).toEqual(["2030-02-11", "2030-03-04"]);
    expect(grid[3][6].future).toBe(true);
    expect(grid[3][2].future).toBe(false);
  });

  it("writes calm lines without a score", () => {
    expect(learnerPracticeLine(summary(0, 0))).toBe("Od ostatniej lekcji nie ma jeszcze zapisanych ćwiczeń.");
    expect(learnerPracticeLine(summary(1, 0))).toBe("Od ostatniej lekcji: 1 dzień z ćwiczeniem.");
    expect(teacherPracticeLine(summary(4, 65))).toBe("Od ostatniej lekcji: 4 dni ćwiczeń, 65 min.");
  });
});
