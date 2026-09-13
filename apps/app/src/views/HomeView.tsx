// Renders the learner calendar with assigned teachers, horizon slots, retained lessons, and learner-attributed cancellations.

import { useCallback, useEffect, useState } from "react";

import {
  bookLesson,
  assignmentDisplayName,
  fetchLearnerCalendar,
  fetchLearnerSlots,
  ownCancellationCount,
  type CalendarResponse,
  type Slot,
} from "../api/scheduling";
import { authCopy } from "../auth/copy";
import { bootstrapAuth, getAuthState, getLearnerDisplayName, logout, subscribe } from "../auth/auth";
import { ApiFeedback, EmptyState, LessonList } from "../components/ScheduleBits";
import { router } from "../router";
import { formatScheduleDate, formatScheduleInstant, groupSlotsByLocalDate } from "../time/schedule";

type LoadState = { status: "loading" | "ready" | "error"; data: CalendarResponse | null; error: unknown };

export function HomeView() {
  const authState = useAuthState();
  const [panel, setPanel] = useState<LoadState>({ status: "loading", data: null, error: null });
  const [slots, setSlots] = useState<Record<string, Slot[]>>({});
  const [busy, setBusy] = useState<string | null>(null);
  const [actionError, setActionError] = useState<unknown>(null);
  const refresh = useCallback(async () => {
    setPanel((current) => ({ ...current, status: "loading", error: null }));
    try {
      const data = await fetchLearnerCalendar();
      const active = data.assignments.filter((assignment) => assignment.active);
      const slotResults = await Promise.all(active.map(async (assignment) => [assignment.id, (await fetchLearnerSlots(assignment.id)).slots] as const));
      setSlots(Object.fromEntries(slotResults));
      setPanel({ status: "ready", data, error: null });
    } catch (error) { setPanel((current) => ({ ...current, status: "error", error })); }
  }, []);
  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => { if (authState.status === "ready" && !authState.record) void router.navigate({ to: "/learners/login" }); }, [authState.status, authState.record]);
  if (authState.status === "error") return <main className="center-shell"><section className="status-card" role="alert"><p className="eyebrow">nutka / sesja</p><h1>{authCopy.unavailable}</h1><button className="secondary-button" onClick={() => void bootstrapAuth()}>{authCopy.retry}</button></section></main>;
  if (!authState.record) return null;
  const title = `Cześć, ${getLearnerDisplayName(authState.record)}.`;
  if (panel.status === "loading" && !panel.data) return <PanelFrame title={title} onLogout={() => void handleLogout(logout)}><p className="loading-state" role="status">Ładowanie kalendarza…</p></PanelFrame>;
  if (panel.status === "error" || !panel.data) return <PanelFrame title={title} onLogout={() => void handleLogout(logout)}><ApiFeedback error={panel.error} onRetry={() => void refresh()} /></PanelFrame>;
  const data = panel.data;
  const teacherNames = new Map(data.assignments.map((assignment) => [assignment.teacher, assignmentDisplayName(assignment, "learner")]));
  async function book(assignmentId: string, slot: Slot) {
    setBusy(`${assignmentId}:${slot.start_at}`); setActionError(null);
    try { await bookLesson(assignmentId, { start_at: slot.start_at }); await refresh(); }
    catch (error) { setActionError(error); } finally { setBusy(null); }
  }
  const activeAssignments = data.assignments.filter((assignment) => assignment.active);
  return <PanelFrame title={title} onLogout={() => void handleLogout(logout)}>
    <ApiFeedback error={actionError} />
    <section className="panel-section" aria-labelledby="learner-summary"><div className="section-heading"><div><p className="eyebrow">nutka / uczeń</p><h2 id="learner-summary">Twój kalendarz</h2></div><Counter value={ownCancellationCount(data.cancellation_counters, "learner")} label="Twoje odwołania" /></div><p className="supporting-copy">Wolne terminy obejmują dziś i kolejne 14 dni. Każda lekcja używa domyślnego czasu przypisania.</p></section>
    <section className="panel-section" aria-labelledby="assigned-heading"><div className="section-heading"><h2 id="assigned-heading">Przypisani nauczyciele</h2></div>{activeAssignments.length === 0 ? <EmptyState>Nie masz jeszcze aktywnych przypisań.</EmptyState> : <div className="assignment-grid">{activeAssignments.map((assignment) => <AssignmentSlots key={assignment.id} assignmentId={assignment.id} label={assignmentDisplayName(assignment, "learner")} duration={assignment.default_duration_minutes} slots={slots[assignment.id] || []} busy={busy} onBook={(slot) => void book(assignment.id, slot)} />)}</div>}</section>
    <section className="panel-section" aria-labelledby="learner-lessons-heading"><div className="section-heading"><h2 id="learner-lessons-heading">Lekcje</h2></div><LessonList lessons={data.lessons} role="learner" counterpartNames={teacherNames} onRefresh={refresh} /></section>
  </PanelFrame>;
}

function AssignmentSlots({ assignmentId, duration, slots, busy, onBook, label }: { assignmentId: string; duration: number; slots: Slot[]; busy: string | null; onBook: (slot: Slot) => void; label: string }) {
  const groups = groupSlotsByLocalDate(slots);
  return <article className="assignment-card"><div className="assignment-heading"><div><p className="eyebrow">Przypisanie</p><h3>{label}</h3></div><span className="duration-badge">{duration} min</span></div><p className="supporting-copy">Wybierz termin z dostępnych godzin.</p>{slots.length === 0 ? <EmptyState>Brak wolnych terminów w horyzoncie.</EmptyState> : <div className="slot-groups">{Array.from(groups.entries()).map(([date, items]) => <div className="slot-group" key={date}><h4>{formatScheduleDate(items[0].start_at)}</h4><div className="slot-grid">{items.map((slot) => <button className="slot-button" key={slot.start_at} disabled={busy === `${assignmentId}:${slot.start_at}`} onClick={() => onBook(slot)}>{busy === `${assignmentId}:${slot.start_at}` ? "Zapisywanie…" : formatScheduleInstant(slot.start_at)}<span>{duration} min</span></button>)}</div></div>)}</div>}</article>;
}

function Counter({ value, label }: { value: number; label: string }) { return <div className="counter"><strong>{value}</strong><span>{label}</span></div>; }

function PanelFrame({ title, onLogout, children }: { title: string; onLogout: () => void; children: React.ReactNode }) {
  return <main className="panel-shell"><header className="panel-header"><div><p className="wordmark">nutka</p><p className="eyebrow">nutka / uczeń</p></div><button className="text-button" onClick={onLogout}>{authCopy.logout}</button></header><section className="panel-hero"><h1>{title}</h1><p>Planuj lekcje z przypisanymi nauczycielami.</p></section>{children}</main>;
}

async function handleLogout(action: () => Promise<unknown>) { await action(); await router.navigate({ to: "/learners/login" }); }

function useAuthState() { const [, rerender] = useState(0); useEffect(() => subscribe(() => rerender((value) => value + 1)), []); return getAuthState(); }
