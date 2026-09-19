import { describe, expect, it } from "vitest";

import type { CalendarResponse, Policy } from "../../api/contracts";
import { rosterRows } from "./roster";

const policy = { lesson_duration_minutes: 45 } as Policy;

function assignment(id: string, active = true) {
  return { id, teacher: "teacher-1", learner: `learner-${id}`, teacher_name: "Ada", learner_name: `Uczeń ${id}`, active };
}

function summary(id: string, overrides: Record<string, unknown> = {}) {
  return { assignment: id, active_plan: "regular_contract", package: null, contract: { id: "c1", status: "active", start_on: "2030-01-01", end_on: "2030-06-30", remaining_monthly_reschedules: 1, remaining_free_cancellations: 2, price_minor: 5000, currency: "PLN" }, payments: { pending: 0, intentionally_unpaid: 0, overdue: 0, credit_minor: 0, currency: "PLN" }, ...overrides };
}

function lesson(id: string, assignmentId: string, startAt: string) {
  return { id, assignment: assignmentId, teacher: "teacher-1", learner: "l", start_at: startAt, end_at: startAt, duration_minutes: 45, plan_type: "regular_contract", schedule_state: "scheduled", unit_price_minor: 5000, currency: "PLN" };
}

function calendar(overrides: Record<string, unknown> = {}): CalendarResponse {
  return { assignments: [], availability_rules: [], availability_exceptions: [], commercial_summaries: [], near_term_lessons: [], later_contract_lessons: [], ...overrides } as unknown as CalendarResponse;
}

describe("roster rows", () => {
  it("reads plan, slot, and settlement from the one calendar payload", () => {
    const data = calendar({
      assignments: [assignment("a1")],
      commercial_summaries: [summary("a1")],
      near_term_lessons: [lesson("l1", "a1", "2030-09-17T15:00:00Z")],
    });
    const [row] = rosterRows(data, policy);
    expect(row.plan).toBe("regular_contract");
    expect(row.weeklySlot).toContain("45 min");
    expect(row.settlement).toBe("settled");
    expect(row.active).toBe(true);
  });

  it("reports an overdue payment ahead of a pending one", () => {
    const data = calendar({
      assignments: [assignment("a1")],
      commercial_summaries: [summary("a1", { payments: { pending: 2, intentionally_unpaid: 0, overdue: 1, credit_minor: 0, currency: "PLN" } })],
    });
    expect(rosterRows(data, policy)[0].settlement).toBe("overdue");
  });

  it("names no plan and no slot when the learner has no commercial summary", () => {
    const data = calendar({ assignments: [assignment("a1", false)] });
    const [row] = rosterRows(data, policy);
    expect(row.plan).toBeNull();
    expect(row.weeklySlot).toBeNull();
    expect(row.active).toBe(false);
  });
});
