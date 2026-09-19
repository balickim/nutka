// Presents the teacher's lessons in the booking horizon and the later contract bookings on their own route.

import { LessonList } from "../../components/ScheduleBits";
import { learnerNames } from "./roster";
import { TeacherShell } from "./TeacherShell";

export function TeacherCalendarView() {
  return <TeacherShell lede="Przejrzyj nadchodzące lekcje, przełóż je lub odwołaj.">
    {({ calendar, policy }) => {
      const names = new Map(calendar.assignments.map((assignment) => [assignment.learner, learnerNames(calendar).get(assignment.id) ?? assignment.learner]));
      return <>
        <section className="panel-section"><h2>Lekcje w najbliższych {policy.booking_horizon_days} dniach</h2><LessonList lessons={calendar.near_term_lessons} role="teacher" policy={policy} commercialSummaries={calendar.commercial_summaries} counterpartNames={names} /></section>
        <section className="panel-section"><h2>Dalsze rezerwacje umowne</h2><LessonList lessons={calendar.later_contract_lessons ?? []} role="teacher" policy={policy} commercialSummaries={calendar.commercial_summaries} counterpartNames={names} /></section>
      </>;
    }}
  </TeacherShell>;
}
