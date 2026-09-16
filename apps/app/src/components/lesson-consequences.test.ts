import { describe, expect, it } from "vitest";

import type { CommercialSummary, Lesson, Policy } from "../api/contracts";
import { learnerChangeIsTimely, lessonChangeNotice } from "./lesson-consequences";

const policy = { learner_change_cutoff_hours: 24, contract_monthly_reschedules: 1, contract_replacement_deadline_days: 30, teacher_cancellation_extension_days: 7 } as Policy;
const base = { start_at: "2030-09-16T12:00:00Z", plan_type: "ad_hoc" } as Lesson;
const now = Date.parse("2030-09-15T12:00:00Z");

describe("lesson change previews", () => {
  it("classifies the exact learner cutoff as timely", () => {
    expect(learnerChangeIsTimely(base, policy, now)).toBe(true);
    expect(learnerChangeIsTimely(base, policy, now + 1)).toBe(false);
  });

  it("explains late package token use and blocks late reschedule copy", () => {
    const lesson = { ...base, plan_type: "package" } as Lesson;
    expect(lessonChangeNotice("cancel", lesson, "learner", policy, undefined, now + 1)).toContain("token zostanie wykorzystany");
    expect(lessonChangeNotice("reschedule", lesson, "learner", policy, undefined, now + 1)).toContain("można tylko odwołać");
  });

  it("uses the contract allowance balance for timely cancellation", () => {
    const lesson = { ...base, plan_type: "regular_contract" } as Lesson;
    const summary = { contract: { remaining_free_cancellations: 2 } } as CommercialSummary;
    expect(lessonChangeNotice("cancel", lesson, "learner", policy, summary, now)).toContain("2 pozostałych");
  });

  it("explains teacher package validity extension from policy", () => {
    const lesson = { ...base, plan_type: "package" } as Lesson;
    expect(lessonChangeNotice("cancel", lesson, "teacher", policy)).toContain("7 dni");
  });
});
