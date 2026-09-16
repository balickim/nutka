// Provides reusable feedback and plan-aware lesson lifecycle controls for teacher and learner panels.

import { useEffect, useState, type FormEvent } from "react";

import { planCopy, outcomeCopy, scheduleStateCopy } from "../api/copy";
import {
  cancelCommercialLesson,
  recordLessonOutcome,
  rescheduleCommercialLesson,
} from "../api/commercial";
import type { CommercialSummary, Lesson, PersonaRole, Policy } from "../api/contracts";
import { getSchedulingErrorMessage } from "../api/scheduling";
import { useLifecycleMutation, useOutcomeMutation } from "../query/commercial";
import { formatScheduleInstant, localInputToUtc, utcToLocalInput } from "../time/schedule";
import { learnerChangeIsTimely, lessonChangeNotice } from "./lesson-consequences";

export function ApiFeedback({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  if (!error) return null;
  return <div className="panel-error" role="alert"><span>{getSchedulingErrorMessage(error)}</span>{onRetry ? <button className="text-button" onClick={onRetry}>Spróbuj ponownie</button> : null}</div>;
}

export function EmptyState({ children }: { children: string }) { return <p className="empty-state">{children}</p>; }

export function canManageLesson(lesson: Lesson, now = Date.now()): boolean {
  return lesson.schedule_state === "scheduled" && Date.parse(lesson.start_at) > now;
}

export function hasUpcomingLessons(lessons: Lesson[], now = Date.now()): boolean {
  return lessons.some((lesson) => canManageLesson(lesson, now));
}

export function lessonParticipantName(lesson: Lesson, role: PersonaRole, names: ReadonlyMap<string, string> = new Map()): string {
  const participantId = role === "teacher" ? lesson.learner : lesson.teacher;
  const name = names.get(participantId)?.trim();
  if (name) return name;
  return participantId.length > 12 ? `${participantId.slice(0, 7)}…${participantId.slice(-4)}` : participantId;
}

export function LessonList({ lessons, role, policy, commercialSummaries = [], counterpartNames }: { lessons: Lesson[]; role: PersonaRole; policy: Policy; commercialSummaries?: CommercialSummary[]; counterpartNames?: ReadonlyMap<string, string> }) {
  const [editing, setEditing] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const lifecycle = useLifecycleMutation();
  const outcome = useOutcomeMutation();
  async function reschedule(lesson: Lesson, start: string) {
    const summary = commercialSummaries.find((item) => item.assignment === lesson.assignment);
    if (!window.confirm(lessonChangeNotice("reschedule", lesson, role, policy, summary))) return;
    setBusy(lesson.id);
    try {
      await lifecycle.mutateAsync({ assignmentId: lesson.assignment, write: () => rescheduleCommercialLesson(role, lesson.id, { start_at: localInputToUtc(start) }) });
      setEditing(null);
    } finally { setBusy(null); }
  }
  async function cancel(lesson: Lesson) {
    const summary = commercialSummaries.find((item) => item.assignment === lesson.assignment);
    if (!window.confirm(lessonChangeNotice("cancel", lesson, role, policy, summary))) return;
    setBusy(lesson.id);
    try { await lifecycle.mutateAsync({ assignmentId: lesson.assignment, write: () => cancelCommercialLesson(role, lesson.id) }); }
    finally { setBusy(null); }
  }
  async function close(lesson: Lesson, value: "completed" | "learner_no_show") {
    setBusy(lesson.id);
    try { await outcome.mutateAsync({ assignmentId: lesson.assignment, write: () => recordLessonOutcome(lesson.id, { outcome: value }) }); }
    finally { setBusy(null); }
  }
  return <div className="lesson-list"><ApiFeedback error={lifecycle.error || outcome.error} />{lessons.length === 0 ? <EmptyState>Nie masz jeszcze żadnych lekcji.</EmptyState> : lessons.map((lesson) => <LessonCard key={lesson.id} lesson={lesson} role={role} policy={policy} summary={commercialSummaries.find((item) => item.assignment === lesson.assignment)} counterpartNames={counterpartNames} editing={editing === lesson.id} busy={busy === lesson.id} onEdit={() => setEditing(editing === lesson.id ? null : lesson.id)} onReschedule={(start) => void reschedule(lesson, start)} onCancel={() => void cancel(lesson)} onOutcome={(value) => void close(lesson, value)} />)}{!hasUpcomingLessons(lessons) && lessons.length > 0 ? <p className="supporting-copy">Brak nadchodzących lekcji.</p> : null}</div>;
}

type LessonCardProps = {
  lesson: Lesson;
  role: PersonaRole;
  policy: Policy;
  summary?: CommercialSummary;
  counterpartNames?: ReadonlyMap<string, string>;
  editing: boolean;
  busy: boolean;
  onEdit: () => void;
  onReschedule: (start: string) => void;
  onCancel: () => void;
  onOutcome: (value: "completed" | "learner_no_show") => void;
};

function LessonCard({ lesson, role, policy, summary, counterpartNames, editing, busy, onEdit, onReschedule, onCancel, onOutcome }: LessonCardProps) {
  const [start, setStart] = useState(() => utcToLocalInput(lesson.start_at));
  const [selectedOutcome, setSelectedOutcome] = useState<"completed" | "learner_no_show">("completed");
  useEffect(() => setStart(utcToLocalInput(lesson.start_at)), [lesson.start_at]);
  const future = canManageLesson(lesson);
  const canReschedule = role === "teacher" || learnerChangeIsTimely(lesson, policy);
  const rescheduleNotice = lessonChangeNotice("reschedule", lesson, role, policy, summary);
  const awaiting = role === "teacher" && lesson.schedule_state === "scheduled" && Date.parse(lesson.end_at) <= Date.now() && (!lesson.outcome || lesson.outcome === "awaiting_outcome");
  function submit(event: FormEvent) { event.preventDefault(); onReschedule(start); }
  return <article className={`lesson-card ${lesson.schedule_state === "scheduled" ? "" : "lesson-cancelled"}`}>
    <div className="lesson-heading"><div><p className="eyebrow">{planCopy[lesson.plan_type]}</p><h3>{lessonParticipantName(lesson, role, counterpartNames)}</h3></div><span className={`status-badge status-${lesson.schedule_state}`}>{scheduleStateCopy[lesson.schedule_state]}</span></div>
    <p className="lesson-meta">{formatScheduleInstant(lesson.start_at)} – {formatScheduleInstant(lesson.end_at)} · {lesson.duration_minutes} min · {(lesson.unit_price_minor / 100).toFixed(0)} {lesson.currency}</p>
    <LessonStatusDetails lesson={lesson} role={role} />
    <FutureLessonActions visible={future} canReschedule={canReschedule} editing={editing} busy={busy} onEdit={onEdit} onCancel={onCancel} />
    {!canReschedule && future ? <p className="supporting-copy">{rescheduleNotice}</p> : null}
    <RescheduleForm visible={editing && future && canReschedule} lessonId={lesson.id} start={start} busy={busy} notice={rescheduleNotice} onStart={setStart} onSubmit={submit} />
    <OutcomeForm visible={awaiting} lessonId={lesson.id} value={selectedOutcome} busy={busy} onChange={setSelectedOutcome} onSubmit={onOutcome} />
  </article>;
}

function LessonStatusDetails({ lesson, role }: { lesson: Lesson; role: PersonaRole }) {
  return <>{lesson.outcome ? <p className="supporting-copy">Wynik: {outcomeCopy[lesson.outcome]}</p> : null}{lesson.cancellation_initiator_role ? <p className="supporting-copy">Odwołana przez {lesson.cancellation_initiator_role === role ? "Ciebie" : "drugą osobę"}.</p> : null}</>;
}

function FutureLessonActions({ visible, canReschedule, editing, busy, onEdit, onCancel }: { visible: boolean; canReschedule: boolean; editing: boolean; busy: boolean; onEdit: () => void; onCancel: () => void }) {
  if (!visible) return null;
  return <div className="lesson-actions">{canReschedule ? <button className="secondary-button" onClick={onEdit} disabled={busy}>{editing ? "Zamknij" : "Przełóż"}</button> : null}<button className="text-button danger-button" onClick={onCancel} disabled={busy}>Odwołaj</button></div>;
}

function RescheduleForm({ visible, lessonId, start, busy, notice, onStart, onSubmit }: { visible: boolean; lessonId: string; start: string; busy: boolean; notice: string; onStart: (value: string) => void; onSubmit: (event: FormEvent) => void }) {
  if (!visible) return null;
  return <form className="inline-form" onSubmit={onSubmit}><label htmlFor={`start-${lessonId}`}>Nowy termin</label><input id={`start-${lessonId}`} type="datetime-local" value={start} onChange={(event) => onStart(event.target.value)} required /><p className="supporting-copy">{notice}</p><button className="primary-button" type="submit" disabled={busy}>Zapisz termin</button></form>;
}

function OutcomeForm({ visible, lessonId, value, busy, onChange, onSubmit }: { visible: boolean; lessonId: string; value: "completed" | "learner_no_show"; busy: boolean; onChange: (value: "completed" | "learner_no_show") => void; onSubmit: (value: "completed" | "learner_no_show") => void }) {
  if (!visible) return null;
  return <form className="inline-form" onSubmit={(event) => { event.preventDefault(); onSubmit(value); }}><label htmlFor={`outcome-${lessonId}`}>Wynik lekcji</label><select id={`outcome-${lessonId}`} value={value} onChange={(event) => onChange(event.target.value as typeof value)}><option value="completed">Zrealizowana</option><option value="learner_no_show">Nieobecność ucznia</option></select><p className="supporting-copy">Domyślny wybór nie zostanie zapisany bez potwierdzenia.</p><button className="primary-button" type="submit" disabled={busy}>Zapisz wynik</button></form>;
}
