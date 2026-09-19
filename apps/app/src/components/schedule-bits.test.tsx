import { describe, expect, it } from "vitest";

import { canManageLesson, hasUpcomingLessons, lessonParticipantName } from "./schedule-bits";
import type { Assignment, Lesson } from "../api/scheduling";

const lesson = (startAt: string, scheduleState: Lesson["schedule_state"] = "scheduled"): Lesson => ({
  id: "lesson-1", teacher: "teacher-1", learner: "learner-1", assignment: "assignment-1",
  start_at: startAt, end_at: "2026-01-15T13:00:00Z", duration_minutes: 45,
  plan_type: "ad_hoc", package_token: null, contract: null, policy_version: "v1",
  unit_price_minor: 8000, currency: "PLN", schedule_state: scheduleState, outcome: null,
  protected_interval: { start_at: startAt, end_at: "2026-01-15T13:05:00Z" },
});

describe("lesson lifecycle controls", () => {
  it("allows controls only for scheduled lessons that have not started", () => {
    const now = Date.parse("2026-01-15T12:00:00Z");
    expect(canManageLesson(lesson("2026-01-15T12:15:00Z"), now)).toBe(true);
    expect(canManageLesson(lesson("2026-01-15T11:45:00Z"), now)).toBe(false);
    expect(canManageLesson(lesson("2026-01-15T12:15:00Z", "cancelled"), now)).toBe(false);
  });

  it("does not call past scheduled history upcoming", () => {
    const now = Date.parse("2026-01-15T12:00:00Z");
    expect(hasUpcomingLessons([lesson("2026-01-15T11:00:00Z")], now)).toBe(false);
    expect(hasUpcomingLessons([lesson("2026-01-15T11:00:00Z"), lesson("2026-01-15T12:15:00Z")], now)).toBe(true);
  });

  it("maps each lesson to its assigned counterpart and falls back to an opaque ID", () => {
    const named = lesson("2026-01-15T12:15:00Z");
    const assignment = { id: "assignment-1", teacher: "teacher-1", learner: "learner-1", learner_name: "Ada", teacher_name: "" } as Assignment;
    expect(lessonParticipantName(named, "teacher", [assignment])).toBe("Ada");
    expect(lessonParticipantName(named, "learner", [])).toBe("teacher-1");
  });
});
