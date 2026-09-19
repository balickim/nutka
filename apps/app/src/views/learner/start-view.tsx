// Answers "what is next" for the selected assignment: the next scheduled lesson and links to the other learner screens.

import { Link } from "@tanstack/react-router";

import type { CalendarResponse, Lesson } from "../../api/contracts";
import { assignmentDisplayName } from "../../api/scheduling";
import { EmptyState } from "../../components/empty-state";
import { formatScheduleInstant } from "../../time/schedule";
import { LearnerShell, type LearnerContext } from "./learner-shell";
import { LatestNote } from "./lessons/lesson-notes";
import { PaymentCard } from "./payments/payment-card";

export function StartView() {
  return <LearnerShell lede="Najbliższa lekcja i to, co warto zrobić przed nią.">{(context) => <Start context={context} />}</LearnerShell>;
}

function Start({ context }: { context: LearnerContext }) {
  const { assignment, calendar, policy } = context;
  const next = nextLesson(calendar, assignment.id);
  const search = { a: assignment.id };
  return <>
    <section className="panel-section">
      <h2>Najbliższa lekcja</h2>
      {next
        ? <div className="next-lesson"><p className="next-lesson-time">{formatScheduleInstant(next.start_at)}</p><p className="supporting-copy">{policy.lesson_duration_minutes} minut · nauczyciel: {assignmentDisplayName(assignment, "learner")}</p><Link className="btn btn-ghost btn-sm" to="/learners/lessons" search={search}>Przełóż lub odwołaj</Link></div>
        : <EmptyState action={<Link className="btn btn-primary btn-sm" to="/learners/lessons" search={search}>Zarezerwuj lekcję</Link>}>Nie masz zaplanowanej lekcji.</EmptyState>}
    </section>
    <LatestNote accountId={context.accountId} assignmentId={assignment.id} />
    <PaymentCard accountId={context.accountId} assignmentId={assignment.id} />
    <section className="panel-section">
      <h2>Materiały od nauczyciela</h2>
      <p className="supporting-copy">Nuty, opracowania i opisy ćwiczeń z lekcji.</p>
      <Link className="btn btn-ghost btn-sm" to="/learners/pieces" search={search}>Otwórz materiały</Link>
    </section>
  </>;
}

export function nextLesson(calendar: CalendarResponse, assignmentId: string, now = Date.now()): Lesson | undefined {
  return calendar.near_term_lessons
    .filter((lesson) => lesson.assignment === assignmentId && lesson.schedule_state === "scheduled" && Date.parse(lesson.start_at) > now)
    .sort((left, right) => left.start_at.localeCompare(right.start_at))[0];
}
