// Derives the teacher roster rows and learner names from the one calendar payload, so no screen fetches per learner.

import { assignmentDisplayName } from "../../api/scheduling";
import type { Assignment, CalendarResponse, CommercialSummary, Lesson, Policy } from "../../api/contracts";
import { formatWeeklySlot } from "../../time/schedule";

export type RosterRow = {
  assignmentId: string;
  learnerName: string;
  plan: CommercialSummary["active_plan"] | null;
  weeklySlot: string | null;
  settlement: "settled" | "pending" | "overdue";
  active: boolean;
};

export function learnerNames(calendar: CalendarResponse): ReadonlyMap<string, string> {
  return new Map(calendar.assignments.map((assignment) => [assignment.id, assignmentDisplayName(assignment, "teacher")]));
}

export function rosterRows(calendar: CalendarResponse, policy: Policy): RosterRow[] {
  const summaries = new Map(calendar.commercial_summaries.map((summary) => [summary.assignment, summary]));
  const lessons = [...calendar.near_term_lessons, ...(calendar.later_contract_lessons ?? [])];
  return calendar.assignments.map((assignment) => row(assignment, summaries.get(assignment.id), lessons, policy));
}

function row(assignment: Assignment, summary: CommercialSummary | undefined, lessons: Lesson[], policy: Policy): RosterRow {
  return {
    assignmentId: assignment.id,
    learnerName: assignmentDisplayName(assignment, "teacher"),
    plan: summary?.active_plan ?? null,
    weeklySlot: summary?.contract ? weeklySlot(assignment.id, lessons, policy) : null,
    settlement: settlementOf(summary),
    active: assignment.active,
  };
}

// The summary names the contract but not its weekday, so the next contract lesson carries the recurring slot.
function weeklySlot(assignmentId: string, lessons: Lesson[], policy: Policy): string | null {
  const next = lessons
    .filter((lesson) => lesson.assignment === assignmentId && lesson.plan_type === "regular_contract" && lesson.schedule_state === "scheduled")
    .sort((left, right) => left.start_at.localeCompare(right.start_at))[0];
  return next ? `${formatWeeklySlot(next.start_at)} · ${policy.lesson_duration_minutes} min` : null;
}

function settlementOf(summary: CommercialSummary | undefined): RosterRow["settlement"] {
  if (!summary) return "settled";
  if (summary.payments.overdue > 0) return "overdue";
  if (summary.payments.pending > 0 || summary.payments.intentionally_unpaid > 0) return "pending";
  return "settled";
}
