// Holds the lesson and plan work of the selected assignment: lessons, teacher notes, booking, plan summary, notice, and history.

import { learnerHelpCopy } from "../../api/copy";
import { ApiFeedback } from "../../components/api-feedback";
import { HelpHeading } from "../../components/help-heading";
import { LessonList } from "../../components/schedule-bits";
import { Skeleton } from "../../components/skeleton";
import { useLearnerSlots } from "../../query/scheduling";
import { LearnerShell, type LearnerContext } from "./learner-shell";
import { NoteList } from "./lessons/lesson-notes";
import { PlanCard } from "./lessons/plan-card";

export function LessonsView() {
  return <LearnerShell lede="Twoje lekcje, rezerwacje i plan zajęć.">{(context) => <Lessons context={context} />}</LearnerShell>;
}

function Lessons({ context }: { context: LearnerContext }) {
  const { accountId, assignment, calendar, policy } = context;
  const slots = useLearnerSlots(accountId, [assignment]);
  const help = learnerHelpCopy(policy);
  const lessons = calendar.near_term_lessons.filter((lesson) => lesson.assignment === assignment.id);
  return <>
    <section className="panel-section"><HelpHeading title="Najbliższe lekcje" help={help.upcoming} /><LessonList lessons={lessons} role="learner" policy={policy} commercialSummaries={calendar.commercial_summaries} assignments={calendar.assignments} /></section>
    <NoteList accountId={accountId} assignmentId={assignment.id} />
    <section className="panel-section"><HelpHeading title="Twój plan i rezerwacje" help={help.plan} /><p className="supporting-copy">Lekcja trwa {policy.lesson_duration_minutes} minut. Przed lekcją i po niej zostaje {policy.participant_buffer_minutes} minut przerwy.</p>
      {slots.error ? <ApiFeedback error={slots.error} onRetry={slots.refetch} /> : slots.pending ? <Skeleton lines={3} label="Ładowanie terminów…" /> : <PlanCard accountId={accountId} assignment={assignment} slots={slots.slotsFor(assignment.id)} policy={policy} />}
    </section>
  </>;
}
