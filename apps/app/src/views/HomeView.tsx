// Renders the learner calendar with assigned teachers, horizon slots, retained lessons, and learner-attributed cancellations.

import { useEffect, useState } from "react";
import { useQueries, useQuery } from "@tanstack/react-query";

import {
  bookLesson,
  assignmentDisplayName,
  ownCancellationCount,
  type Assignment,
  type CalendarResponse,
  type Slot,
} from "../api/scheduling";
import { authCopy } from "../auth/copy";
import { authenticatedRecord, personaDisplayName, usePersonaLogout, usePersonaSession } from "../auth/session";
import { ApiFeedback, EmptyState, LessonList } from "../components/ScheduleBits";
import { ignoreWriteRejection, learnerCalendarQuery, useLearnerSlots, useLessonWrite } from "../query/scheduling";
import { router } from "../router";
import { formatScheduleDate, formatScheduleInstant, groupSlotsByLocalDate } from "../time/schedule";

export function HomeView() {
  const session = usePersonaSession("learner");
  const record = authenticatedRecord(session.data);
  const accountId = record?.id;
  const calendar = useQuery(learnerCalendarQuery(accountId));
  const activeAssignments = activeOf(calendar.data);
  const slots = useLearnerSlots(accountId, activeAssignments);
  const logout = usePersonaLogout("learner");
  const booking = useLessonWrite();
  const [busy, setBusy] = useState<string | null>(null);
  const unauthenticated = session.data?.kind === "unauthenticated";
  useEffect(() => { if (unauthenticated) void router.navigate({ to: "/learners/login" }); }, [unauthenticated]);
  if (session.isError) return <SessionUnavailable onRetry={() => void session.refetch()} />;
  if (!record) return null;
  const data = calendar.data;
  const frame = (children: React.ReactNode) => <PanelFrame title={`Cześć, ${personaDisplayName(record)}.`} onLogout={() => void handleLogout()}>{children}</PanelFrame>;
  async function handleLogout() { await logout.mutateAsync(); await router.navigate({ to: "/learners/login" }); }
  function retry() { void calendar.refetch(); slots.refetch(); }
  async function book(assignmentId: string, slot: Slot) {
    setBusy(`${assignmentId}:${slot.start_at}`);
    await ignoreWriteRejection(booking.mutateAsync(() => bookLesson(assignmentId, { start_at: slot.start_at })));
    setBusy(null);
  }
  const panelError = calendar.error || slots.error;
  if (panelError) return frame(<ApiFeedback error={panelError} onRetry={retry} />);
  if (!data || slots.pending) return frame(<p className="loading-state" role="status">Ładowanie kalendarza…</p>);
  const teacherNames = new Map(data.assignments.map((assignment) => [assignment.teacher, assignmentDisplayName(assignment, "learner")]));
  return frame(<>
    <ApiFeedback error={booking.error} />
    <section className="panel-section" aria-labelledby="learner-summary"><div className="section-heading"><div><p className="eyebrow">nutka / uczeń</p><h2 id="learner-summary">Twój kalendarz</h2></div><Counter value={ownCancellationCount(data.cancellation_counters, "learner")} label="Twoje odwołania" /></div><p className="supporting-copy">Wolne terminy obejmują dziś i kolejne 14 dni. Każda lekcja używa domyślnego czasu przypisania.</p></section>
    <section className="panel-section" aria-labelledby="assigned-heading"><div className="section-heading"><h2 id="assigned-heading">Przypisani nauczyciele</h2></div><AssignmentGrid assignments={activeAssignments} slotsFor={slots.slotsFor} busy={busy} onBook={book} /></section>
    <section className="panel-section" aria-labelledby="learner-lessons-heading"><div className="section-heading"><h2 id="learner-lessons-heading">Lekcje</h2></div><LessonList lessons={data.lessons} role="learner" counterpartNames={teacherNames} /></section>
  </>);
}

function activeOf(calendar: CalendarResponse | undefined): Assignment[] {
  if (!calendar) return [];
  return calendar.assignments.filter((assignment) => assignment.active);
}

function SessionUnavailable({ onRetry }: { onRetry: () => void }) {
  return <main className="center-shell"><section className="status-card" role="alert"><p className="eyebrow">nutka / sesja</p><h1>{authCopy.unavailable}</h1><button className="secondary-button" onClick={onRetry}>{authCopy.retry}</button></section></main>;
}

function AssignmentGrid({ assignments, slotsFor, busy, onBook }: { assignments: Assignment[]; slotsFor: (assignmentId: string) => Slot[]; busy: string | null; onBook: (assignmentId: string, slot: Slot) => Promise<void> }) {
  if (assignments.length === 0) return <EmptyState>Nie masz jeszcze aktywnych przypisań.</EmptyState>;
  return <div className="assignment-grid">{assignments.map((assignment) => <AssignmentSlots key={assignment.id} assignmentId={assignment.id} label={assignmentDisplayName(assignment, "learner")} duration={assignment.default_duration_minutes} slots={slotsFor(assignment.id)} busy={busy} onBook={(slot) => void onBook(assignment.id, slot)} />)}</div>;
}

function AssignmentSlots({ assignmentId, duration, slots, busy, onBook, label }: { assignmentId: string; duration: number; slots: Slot[]; busy: string | null; onBook: (slot: Slot) => void; label: string }) {
  const groups = groupSlotsByLocalDate(slots);
  return <article className="assignment-card"><div className="assignment-heading"><div><p className="eyebrow">Przypisanie</p><h3>{label}</h3></div><span className="duration-badge">{duration} min</span></div><p className="supporting-copy">Wybierz termin z dostępnych godzin.</p>{slots.length === 0 ? <EmptyState>Brak wolnych terminów w horyzoncie.</EmptyState> : <div className="slot-groups">{Array.from(groups.entries()).map(([date, items]) => <div className="slot-group" key={date}><h4>{formatScheduleDate(items[0].start_at)}</h4><div className="slot-grid">{items.map((slot) => <button className="slot-button" key={slot.start_at} disabled={busy === `${assignmentId}:${slot.start_at}`} onClick={() => onBook(slot)}>{busy === `${assignmentId}:${slot.start_at}` ? "Zapisywanie…" : formatScheduleInstant(slot.start_at)}<span>{duration} min</span></button>)}</div></div>)}</div>}</article>;
}

function Counter({ value, label }: { value: number; label: string }) { return <div className="counter"><strong>{value}</strong><span>{label}</span></div>; }

function PanelFrame({ title, onLogout, children }: { title: string; onLogout: () => void; children: React.ReactNode }) {
  return <main className="panel-shell"><header className="panel-header"><div><p className="wordmark">nutka</p><p className="eyebrow">nutka / uczeń</p></div><button className="text-button" onClick={onLogout}>{authCopy.logout}</button></header><section className="panel-hero"><h1>{title}</h1><p>Planuj lekcje z przypisanymi nauczycielami.</p></section>{children}</main>;
}
