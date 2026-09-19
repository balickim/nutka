// Presents the teacher's lessons in the booking horizon and the later contract bookings on their own route.

import { LessonList } from "../../components/ScheduleBits";
import { TeacherShell } from "./TeacherShell";

export function TeacherCalendarView() {
  return <TeacherShell lede="Przejrzyj nadchodzące lekcje, przełóż je lub odwołaj.">
    {({ calendar, policy }) => <>
      <section className="panel-section"><h2>Lekcje w najbliższych {policy.booking_horizon_days} dniach</h2><LessonList lessons={calendar.near_term_lessons} role="teacher" policy={policy} commercialSummaries={calendar.commercial_summaries} assignments={calendar.assignments} /></section>
      <section className="panel-section"><h2>Dalsze rezerwacje umowne</h2><LessonList lessons={calendar.later_contract_lessons ?? []} role="teacher" policy={policy} commercialSummaries={calendar.commercial_summaries} assignments={calendar.assignments} /></section>
    </>}
  </TeacherShell>;
}
