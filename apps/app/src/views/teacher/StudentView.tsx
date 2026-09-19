// Presents one learner's commercial workspace, loading that learner's data only and refusing a foreign assignment.

import { useState } from "react";
import { useParams } from "@tanstack/react-router";

import { assignmentDisplayName } from "../../api/scheduling";
import type { Assignment, CalendarResponse, Lesson, Policy } from "../../api/contracts";
import { EmptyState } from "../../components/EmptyState";
import { HistoryTab } from "./student/HistoryTab";
import { OverviewTab } from "./student/OverviewTab";
import { PlanTab } from "./student/PlanTab";
import { TeacherShell } from "./TeacherShell";

const tabs = [
  { id: "overview", label: "Przegląd" },
  { id: "plan", label: "Plan" },
  { id: "history", label: "Historia" },
] as const;

type TabId = (typeof tabs)[number]["id"];

export function StudentView() {
  const { assignmentId } = useParams({ from: "/teachers/students/$assignmentId" });
  return <TeacherShell lede="Stan planu, rozliczeń i historii jednego ucznia.">
    {(context) => {
      const assignment = context.calendar.assignments.find((item) => item.id === assignmentId);
      if (!assignment) return <NotYours />;
      return <Student accountId={context.accountId} assignment={assignment} policy={context.policy} adHocLessons={adHocOf(context.calendar, assignment.id)} />;
    }}
  </TeacherShell>;
}

function NotYours() {
  return <section className="panel-section" role="alert"><h2>Brak dostępu</h2><EmptyState>Ten uczeń nie jest przypisany do Ciebie.</EmptyState></section>;
}

function adHocOf(calendar: CalendarResponse, assignmentId: string): Lesson[] {
  return calendar.near_term_lessons.filter((lesson) => lesson.assignment === assignmentId && lesson.plan_type === "ad_hoc" && lesson.schedule_state === "scheduled");
}

function Student({ accountId, assignment, policy, adHocLessons }: { accountId: string; assignment: Assignment; policy: Policy; adHocLessons: Lesson[] }) {
  const [tab, setTab] = useState<TabId>("overview");
  return <section className="panel-section">
    <div className="assignment-heading"><div><p className="eyebrow">Uczeń</p><h2>{assignmentDisplayName(assignment, "teacher")}</h2></div><span className="duration-badge">{policy.lesson_duration_minutes} min</span></div>
    <div className="tab-bar" role="tablist">{tabs.map((item) => <button key={item.id} role="tab" type="button" aria-selected={tab === item.id} className={tab === item.id ? "active" : ""} onClick={() => setTab(item.id)}>{item.label}</button>)}</div>
    {tab === "overview" ? <OverviewTab accountId={accountId} assignment={assignment} /> : null}
    {tab === "plan" ? <PlanTab accountId={accountId} assignment={assignment} policy={policy} adHocLessons={adHocLessons} /> : null}
    {tab === "history" ? <HistoryTab accountId={accountId} assignmentId={assignment.id} /> : null}
  </section>;
}
