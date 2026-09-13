// Renders teacher lessons, assignments, recurring availability, exceptions, and teacher-attributed cancellation totals.

import { useEffect, useState, type FormEvent, type ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import {
  createAvailabilityException,
  createAvailabilityRule,
  deleteAvailabilityException,
  deleteAvailabilityRule,
  assignmentDisplayName,
  ownCancellationCount,
  updateAvailabilityException,
  updateAvailabilityRule,
  updateTeacherAssignment,
  type AvailabilityException,
  type AvailabilityRule,
  type CalendarResponse,
} from "../api/scheduling";
import { ApiFeedback, EmptyState, LessonList } from "../components/ScheduleBits";
import { authenticatedRecord, personaDisplayName, usePersonaLogout, usePersonaSession } from "../auth/session";
import { ignoreWriteRejection, teacherCalendarQuery, useAssignmentWrite, useAvailabilityWrite } from "../query/scheduling";
import { router } from "../router";
import { formatScheduleInstant, futureExceptionDraft, localInputToUtc, utcToLocalInput, weekdayLabels } from "../time/schedule";

type TeacherViewProps = { availability?: boolean };
export type AvailabilityWrite = (write: () => Promise<unknown>) => Promise<void>;
export type AssignmentWrite = (assignmentId: string, write: () => Promise<unknown>) => Promise<void>;

export function TeacherView({ availability = false }: TeacherViewProps) {
  const session = usePersonaSession("teacher");
  const record = authenticatedRecord(session.data);
  const calendar = useQuery(teacherCalendarQuery(record?.id));
  const logout = usePersonaLogout("teacher");
  const availabilityWrite = useAvailabilityWrite();
  const assignmentWrite = useAssignmentWrite();
  const unauthenticated = session.data?.kind === "unauthenticated";
  useEffect(() => { if (unauthenticated) void router.navigate({ to: "/teachers/login" }); }, [unauthenticated]);
  if (session.isError) return <main className="center-shell"><section className="status-card" role="alert"><h1>Nie możemy teraz sprawdzić sesji.</h1><button className="secondary-button" onClick={() => void session.refetch()}>Spróbuj ponownie</button></section></main>;
  if (!record) return null;
  const title = `Cześć, ${personaDisplayName(record)}.`;
  const frame = (children: ReactNode) => <PanelFrame title={title} availability={availability} onLogout={() => void handleLogout()}>{children}</PanelFrame>;
  async function handleLogout() { await logout.mutateAsync(); await router.navigate({ to: "/teachers/login" }); }
  if (calendar.error) return frame(<ApiFeedback error={calendar.error} onRetry={() => void calendar.refetch()} />);
  if (!calendar.data) return frame(<p className="loading-state" role="status">Ładowanie kalendarza…</p>);
  const data = calendar.data;
  const onAvailability: AvailabilityWrite = (write) => ignoreWriteRejection(availabilityWrite.mutateAsync(write));
  const onAssignment: AssignmentWrite = (assignmentId, write) => ignoreWriteRejection(assignmentWrite.mutateAsync({ assignmentId, write }));
  return frame(<>
    <ApiFeedback error={availabilityWrite.error || assignmentWrite.error} />
    {availability
      ? <AvailabilityPanel data={data} timezone={record.timezone} onMutate={onAvailability} />
      : <TeacherDashboard data={data} onMutate={onAssignment} />}
  </>);
}

function TeacherDashboard({ data, onMutate }: { data: CalendarResponse; onMutate: AssignmentWrite }) {
  const learnerNames = new Map(data.assignments.map((assignment) => [assignment.learner, assignmentDisplayName(assignment, "teacher")]));
  return <><section className="panel-section" aria-labelledby="teacher-summary"><div className="section-heading"><div><p className="eyebrow">nutka / nauczyciel</p><h2 id="teacher-summary">Twój kalendarz</h2></div><Counter value={ownCancellationCount(data.cancellation_counters, "teacher")} label="Twoje odwołania" /></div><p className="supporting-copy">Zarządzaj przypisaniami i przyszłymi lekcjami. Zmiana czasu dotyczy tylko Ciebie jako nauczyciela.</p><p><Link className="secondary-button inline-link" to="/teachers/availability">Edytuj dostępność</Link></p></section><AssignmentPanel assignments={data.assignments} onMutate={onMutate} /><section className="panel-section" aria-labelledby="teacher-lessons-heading"><div className="section-heading"><h2 id="teacher-lessons-heading">Lekcje</h2></div><LessonList lessons={data.lessons} role="teacher" counterpartNames={learnerNames} /></section></>;
}

function AssignmentPanel({ assignments, onMutate }: { assignments: CalendarResponse["assignments"]; onMutate: AssignmentWrite }) {
  return <section className="panel-section" aria-labelledby="assignments-heading"><div className="section-heading"><h2 id="assignments-heading">Przypisani uczniowie</h2></div>{assignments.length === 0 ? <EmptyState>Nie masz jeszcze przypisanych uczniów.</EmptyState> : <div className="assignment-list">{assignments.map((assignment) => <AssignmentRow key={assignment.id} assignment={assignment} onSave={(body) => onMutate(assignment.id, () => updateTeacherAssignment(assignment.id, body))} />)}</div>}</section>;
}

function AssignmentRow({ assignment, onSave }: { assignment: CalendarResponse["assignments"][number]; onSave: (body: { active?: boolean; default_duration_minutes?: number }) => Promise<void> }) {
  const [duration, setDuration] = useState(String(assignment.default_duration_minutes));
  const [busy, setBusy] = useState(false);
  async function save() { setBusy(true); try { await onSave({ default_duration_minutes: Number(duration) }); } finally { setBusy(false); } }
  async function toggle() { setBusy(true); try { await onSave({ active: !assignment.active }); } finally { setBusy(false); } }
  return <article className="assignment-row"><div><p className="eyebrow">Uczeń</p><h3>{assignmentDisplayName(assignment, "teacher")}</h3><p className="supporting-copy">{assignment.active ? "Aktywne przypisanie" : "Nieaktywne przypisanie"} · Domyślny czas: {assignment.default_duration_minutes} min</p></div><div className="assignment-controls"><label htmlFor={`assignment-duration-${assignment.id}`}>Czas (min)</label><input id={`assignment-duration-${assignment.id}`} type="number" min="15" step="15" value={duration} onChange={(event) => setDuration(event.target.value)} /><button className="secondary-button" onClick={() => void save()} disabled={busy}>{busy ? "Zapisywanie…" : "Zapisz"}</button><button className="text-button" onClick={() => void toggle()} disabled={busy}>{assignment.active ? "Dezaktywuj" : "Aktywuj"}</button></div></article>;
}

function AvailabilityPanel({ data, timezone, onMutate }: { data: CalendarResponse; timezone?: string; onMutate: AvailabilityWrite }) {
  return <><section className="panel-section" aria-labelledby="availability-summary"><div className="section-heading"><div><p className="eyebrow">nutka / dostępność</p><h2 id="availability-summary">Tygodniowy plan</h2></div><span className="timezone-badge">Reguły: {timezone || "Europe/Warsaw"}</span></div><p className="supporting-copy">Reguły powtarzają się bez daty końcowej w strefie nauczyciela. Wyjątki i lekcje są konkretne i pokazane w strefie lokalnej przeglądarki.</p><RuleForm onCreate={(body) => onMutate(() => createAvailabilityRule(body))} /><RuleList rules={data.availability_rules} onMutate={onMutate} /></section><section className="panel-section" aria-labelledby="exceptions-heading"><div className="section-heading"><h2 id="exceptions-heading">Wyjątki dat</h2></div><ExceptionForm onCreate={(body) => onMutate(() => createAvailabilityException(body))} /><ExceptionList exceptions={data.availability_exceptions} onMutate={onMutate} /></section><section className="panel-section" aria-labelledby="availability-lessons-heading"><div className="section-heading"><h2 id="availability-lessons-heading">Przyszłe lekcje</h2></div><LessonList lessons={data.lessons} role="teacher" /></section></>;
}

function RuleForm({ onCreate }: { onCreate: (body: Omit<AvailabilityRule, "id" | "teacher">) => Promise<void> }) {
  const [weekday, setWeekday] = useState("1"); const [start, setStart] = useState("16:00"); const [end, setEnd] = useState("20:00"); const [busy, setBusy] = useState(false);
  async function submit(event: FormEvent) { event.preventDefault(); setBusy(true); try { await onCreate({ weekday: Number(weekday), start_time: start, end_time: end, enabled: true }); } finally { setBusy(false); } }
  return <form className="availability-form" onSubmit={(event) => void submit(event)}><div><label htmlFor="rule-weekday">Dzień tygodnia</label><select id="rule-weekday" value={weekday} onChange={(event) => setWeekday(event.target.value)}>{weekdayLabels.map((label, index) => <option key={label} value={index}>{label}</option>)}</select></div><div><label htmlFor="rule-start">Od</label><input id="rule-start" type="time" value={start} onChange={(event) => setStart(event.target.value)} required /></div><div><label htmlFor="rule-end">Do</label><input id="rule-end" type="time" value={end} onChange={(event) => setEnd(event.target.value)} required /></div><button className="primary-button" type="submit" disabled={busy}>{busy ? "Dodawanie…" : "Dodaj regułę"}</button></form>;
}

function RuleList({ rules, onMutate }: { rules: AvailabilityRule[]; onMutate: AvailabilityWrite }) {
  if (rules.length === 0) return <EmptyState>Brak reguł. Tydzień jest domyślnie niedostępny.</EmptyState>;
  return <div className="rule-list">{rules.map((rule) => <RuleRow key={rule.id} rule={rule} onMutate={onMutate} />)}</div>;
}

function RuleRow({ rule, onMutate }: { rule: AvailabilityRule; onMutate: AvailabilityWrite }) {
  const [editing, setEditing] = useState(false); const [weekday, setWeekday] = useState(String(rule.weekday)); const [start, setStart] = useState(rule.start_time); const [end, setEnd] = useState(rule.end_time); const [busy, setBusy] = useState(false);
  async function save(body: Partial<Omit<AvailabilityRule, "id" | "teacher">>) { setBusy(true); try { await onMutate(() => updateAvailabilityRule(rule.id, body)); } finally { setBusy(false); } }
  return <article className="rule-row"><div>{editing ? <div className="compact-fields"><select aria-label="Dzień reguły" value={weekday} onChange={(event) => setWeekday(event.target.value)}>{weekdayLabels.map((label, index) => <option key={label} value={index}>{label}</option>)}</select><input aria-label="Początek reguły" type="time" value={start} onChange={(event) => setStart(event.target.value)} /><input aria-label="Koniec reguły" type="time" value={end} onChange={(event) => setEnd(event.target.value)} /></div> : <><strong>{weekdayLabels[rule.weekday]}</strong><span>{rule.start_time}–{rule.end_time}</span></>}</div><div className="row-actions">{editing ? <button className="secondary-button" onClick={() => void save({ weekday: Number(weekday), start_time: start, end_time: end })} disabled={busy}>Zapisz</button> : <button className="text-button" onClick={() => setEditing(true)}>Edytuj</button>}<button className="text-button" onClick={() => void save({ enabled: !rule.enabled })} disabled={busy}>{rule.enabled ? "Wyłącz" : "Włącz"}</button><button className="text-button danger-button" onClick={() => void onMutate(() => deleteAvailabilityRule(rule.id))} disabled={busy}>Usuń</button></div></article>;
}

function ExceptionForm({ onCreate }: { onCreate: (body: { start_at: string; end_at: string; kind: "available" | "unavailable"; note?: string }) => Promise<void> }) {
  const draft = futureExceptionDraft(); const [start, setStart] = useState(draft.start); const [end, setEnd] = useState(draft.end); const [kind, setKind] = useState<"available" | "unavailable">("unavailable"); const [note, setNote] = useState(""); const [busy, setBusy] = useState(false); const [error, setError] = useState<string | null>(null);
  async function submit(event: FormEvent) { event.preventDefault(); setError(null); setBusy(true); try { await onCreate({ start_at: localInputToUtc(start), end_at: localInputToUtc(end), kind, ...(note.trim() ? { note: note.trim() } : {}) }); setNote(""); } catch (nextError) { setError(nextError instanceof Error ? nextError.message : "Wybierz poprawny przedział."); } finally { setBusy(false); } }
  return <form className="exception-form" onSubmit={(event) => void submit(event)}><p className="supporting-copy">Nowy wyjątek w strefie lokalnej przeglądarki.</p><div><label htmlFor="exception-start">Od</label><input id="exception-start" type="datetime-local" value={start} onChange={(event) => setStart(event.target.value)} required /></div><div><label htmlFor="exception-end">Do</label><input id="exception-end" type="datetime-local" value={end} onChange={(event) => setEnd(event.target.value)} required /></div><div><label htmlFor="exception-kind">Rodzaj</label><select id="exception-kind" value={kind} onChange={(event) => setKind(event.target.value as typeof kind)}><option value="unavailable">Niedostępny</option><option value="available">Dostępny</option></select></div><div><label htmlFor="exception-note">Notatka (opcjonalnie)</label><input id="exception-note" value={note} onChange={(event) => setNote(event.target.value)} /></div>{error ? <p className="form-error" role="alert">{error}</p> : null}<button className="primary-button" type="submit" disabled={busy}>{busy ? "Dodawanie…" : "Dodaj wyjątek"}</button></form>;
}

function ExceptionList({ exceptions, onMutate }: { exceptions: AvailabilityException[]; onMutate: AvailabilityWrite }) {
  if (exceptions.length === 0) return <EmptyState>Brak wyjątków dat.</EmptyState>;
  return <div className="exception-list">{exceptions.map((exception) => <ExceptionRow key={exception.id} exception={exception} onMutate={onMutate} />)}</div>;
}

function ExceptionRow({ exception, onMutate }: { exception: AvailabilityException; onMutate: AvailabilityWrite }) {
  const [editing, setEditing] = useState(false); const [start, setStart] = useState(() => utcToLocalInput(exception.start_at)); const [end, setEnd] = useState(() => utcToLocalInput(exception.end_at)); const [kind, setKind] = useState(exception.kind); const [note, setNote] = useState(exception.note || ""); const [busy, setBusy] = useState(false); const [error, setError] = useState<string | null>(null);
  async function save() { setError(null); setBusy(true); try { await onMutate(() => updateAvailabilityException(exception.id, { start_at: localInputToUtc(start), end_at: localInputToUtc(end), kind, note })); setEditing(false); } catch (nextError) { setError(nextError instanceof Error ? nextError.message : "Nie udało się zapisać wyjątku."); } finally { setBusy(false); } }
  return <article className="exception-row">{editing ? <div className="exception-edit"><input aria-label="Początek wyjątku" type="datetime-local" value={start} onChange={(event) => setStart(event.target.value)} /><input aria-label="Koniec wyjątku" type="datetime-local" value={end} onChange={(event) => setEnd(event.target.value)} /><select aria-label="Rodzaj wyjątku" value={kind} onChange={(event) => setKind(event.target.value as typeof kind)}><option value="unavailable">Niedostępny</option><option value="available">Dostępny</option></select><input aria-label="Notatka wyjątku" value={note} onChange={(event) => setNote(event.target.value)} />{error ? <p className="form-error" role="alert">{error}</p> : null}</div> : <div><strong>{exception.kind === "unavailable" ? "Niedostępny" : "Dostępny"}</strong><p>{formatScheduleInstant(exception.start_at)} – {formatScheduleInstant(exception.end_at)}</p>{exception.note ? <span className="supporting-copy">{exception.note}</span> : null}</div>}<div className="row-actions">{editing ? <button className="secondary-button" onClick={() => void save()} disabled={busy}>Zapisz</button> : <button className="text-button" onClick={() => setEditing(true)}>Edytuj</button>}<button className="text-button danger-button" onClick={() => void onMutate(() => deleteAvailabilityException(exception.id))} disabled={busy}>Usuń</button></div></article>;
}

function Counter({ value, label }: { value: number; label: string }) { return <div className="counter"><strong>{value}</strong><span>{label}</span></div>; }
function PanelFrame({ title, availability, onLogout, children }: { title: string; availability: boolean; onLogout: () => void; children: ReactNode }) { return <main className="panel-shell"><header className="panel-header"><div><p className="wordmark">nutka</p><p className="eyebrow">nutka / nauczyciel</p></div><nav className="panel-nav" aria-label="Nawigacja nauczyciela"><Link className={!availability ? "active" : ""} to="/teachers">Lekcje</Link><Link className={availability ? "active" : ""} to="/teachers/availability">Dostępność</Link></nav><button className="text-button" onClick={onLogout}>Wyloguj się</button></header><section className="panel-hero"><h1>{title}</h1><p>{availability ? "Ustal godziny, w których uczniowie mogą rezerwować lekcje." : "Zobacz przypisania i zarządzaj przyszłymi lekcjami."}</p></section>{children}</main>; }
