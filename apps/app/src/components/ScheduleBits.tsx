// Provides reusable scheduling feedback and lesson lifecycle controls for teacher and learner panels.

import { useEffect, useState, type FormEvent } from "react";

import { cancelLesson, getSchedulingErrorMessage, rescheduleLesson, type Lesson, type PersonaRole } from "../api/scheduling";
import { useLessonWrite } from "../query/scheduling";
import { formatScheduleInstant, localInputToUtc, utcToLocalInput } from "../time/schedule";

export function ApiFeedback({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  if (!error) return null;
  const message = getSchedulingErrorMessage(error);
  return <div className="panel-error" role="alert"><span>{message}</span>{onRetry ? <button className="text-button" onClick={onRetry}>Spróbuj ponownie</button> : null}</div>;
}

export function EmptyState({ children }: { children: string }) { return <p className="empty-state">{children}</p>; }

export function canManageLesson(lesson: Lesson, now = Date.now()): boolean { return lesson.status === "scheduled" && Date.parse(lesson.start_at) > now; }
export function hasUpcomingLessons(lessons: Lesson[], now = Date.now()): boolean { return lessons.some((lesson) => canManageLesson(lesson, now)); }
export function lessonParticipantName(lesson: Lesson, role: PersonaRole, names: ReadonlyMap<string, string> = new Map()): string {
  const participantId = role === "teacher" ? lesson.learner : lesson.teacher;
  const name = names.get(participantId)?.trim();
  if (name) return name;
  return participantId.length > 12 ? `${participantId.slice(0, 7)}…${participantId.slice(-4)}` : participantId;
}

export function LessonList({ lessons, role, counterpartNames }: { lessons: Lesson[]; role: PersonaRole; counterpartNames?: ReadonlyMap<string, string> }) {
  const [editing, setEditing] = useState<string | null>(null); const [busy, setBusy] = useState<string | null>(null);
  const lessonWrite = useLessonWrite();
  async function mutate(lesson: Lesson, action: "reschedule" | "cancel", start?: string, duration?: number) {
    setBusy(lesson.id);
    const write = action === "cancel"
      ? () => cancelLesson(role, lesson.id)
      : () => rescheduleLesson(role, lesson.id, { start_at: localInputToUtc(start || ""), ...(duration ? { duration_minutes: duration } : {}) });
    try { await lessonWrite.mutateAsync(write); setEditing(null); }
    catch { /* The mutation error renders the stable Polish message. */ }
    finally { setBusy(null); }
  }
  return <div className="lesson-list"><ApiFeedback error={lessonWrite.error} />{lessons.length === 0 ? <EmptyState>Nie masz jeszcze żadnych lekcji.</EmptyState> : lessons.map((lesson) => <LessonCard key={lesson.id} lesson={lesson} role={role} counterpartNames={counterpartNames} editing={editing === lesson.id} busy={busy === lesson.id} onEdit={() => setEditing(editing === lesson.id ? null : lesson.id)} onReschedule={(start, duration) => void mutate(lesson, "reschedule", start, duration)} onCancel={() => void mutate(lesson, "cancel")} />)}{!hasUpcomingLessons(lessons) && lessons.length > 0 ? <p className="supporting-copy">Brak nadchodzących lekcji. Zachowana historia pozostaje poniżej.</p> : null}</div>;
}

function LessonCard({ lesson, role, counterpartNames, editing, busy, onEdit, onReschedule, onCancel }: { lesson: Lesson; role: PersonaRole; counterpartNames?: ReadonlyMap<string, string>; editing: boolean; busy: boolean; onEdit: () => void; onReschedule: (start: string, duration?: number) => void; onCancel: () => void }) {
  const [start, setStart] = useState(() => utcToLocalInput(lesson.start_at)); const [duration, setDuration] = useState(String(lesson.duration_minutes)); const [formError, setFormError] = useState<unknown>(null);
  useEffect(() => { setStart(utcToLocalInput(lesson.start_at)); setDuration(String(lesson.duration_minutes)); }, [lesson.start_at, lesson.duration_minutes]);
  const scheduled = lesson.status === "scheduled"; const future = canManageLesson(lesson);
  function submit(event: FormEvent) { event.preventDefault(); setFormError(null); try { onReschedule(start, role === "teacher" ? Number(duration) : undefined); } catch (error) { setFormError(error); } }
  return <article className={`lesson-card ${scheduled ? "" : "lesson-cancelled"}`}><div className="lesson-heading"><div><p className="eyebrow">{role === "teacher" ? "lekcja / uczeń" : "lekcja / nauczyciel"}</p><h3>{lessonParticipantName(lesson, role, counterpartNames)}</h3></div><span className={`status-badge ${scheduled ? "status-scheduled" : "status-cancelled"}`}>{scheduled ? "Zaplanowana" : "Odwołana"}</span></div><p className="lesson-meta">{formatScheduleInstant(lesson.start_at)} – {formatScheduleInstant(lesson.end_at)} · {lesson.duration_minutes} min</p><ApiFeedback error={formError} />{!scheduled && lesson.cancellation_initiator_role ? <p className="supporting-copy">Odwołana przez {lesson.cancellation_initiator_role === role ? "Ciebie" : "drugą osobę"}.</p> : null}{scheduled && future ? <div className="lesson-actions"><button className="secondary-button" onClick={onEdit} disabled={busy}>{editing ? "Zamknij" : "Przełóż"}</button><button className="text-button danger-button" onClick={onCancel} disabled={busy}>Odwołaj</button></div> : scheduled ? <p className="supporting-copy">Lekcja rozpoczęta lub zakończona. Zmiany są zamknięte.</p> : null}{editing && scheduled && future ? <form className="inline-form" onSubmit={submit}><label htmlFor={`start-${lesson.id}`}>Nowy termin (strefa lokalna)</label><input id={`start-${lesson.id}`} type="datetime-local" value={start} onChange={(event) => setStart(event.target.value)} required />{role === "teacher" ? <><label htmlFor={`duration-${lesson.id}`}>Czas trwania (minuty)</label><input id={`duration-${lesson.id}`} type="number" min="15" step="15" value={duration} onChange={(event) => setDuration(event.target.value)} required /></> : null}<button className="primary-button" type="submit" disabled={busy}>{busy ? "Zapisywanie…" : "Zapisz termin"}</button></form> : null}</article>;
}
