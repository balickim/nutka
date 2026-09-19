// Provides plan-aware lesson lifecycle controls shared by the teacher and learner panels.

import { useEffect, useState, type FormEvent } from "react";

import { planCopy, outcomeCopy, scheduleStateCopy } from "../api/copy";
import {
  cancelCommercialLesson,
  recordLessonOutcome,
  rescheduleCommercialLesson,
} from "../api/commercial";
import type { Assignment, CommercialSummary, Lesson, PersonaRole, Policy } from "../api/contracts";
import { assignmentDisplayName, opaqueIdLabel } from "../api/scheduling";
import { useLifecycleMutation, useOutcomeMutation } from "../query/commercial";
import { formatMoney } from "../money";
import { formatScheduleInstant, localInputToUtc, utcToLocalInput } from "../time/schedule";
import { ApiFeedback } from "./api-feedback";
import { ConfirmDialog } from "./confirm-dialog";
import { useToast } from "./toast";
import { EmptyState } from "./empty-state";
import { learnerChangeIsTimely, lessonChangeNotice } from "./lesson-consequences";

export function canManageLesson(lesson: Lesson, now = Date.now()): boolean {
  return lesson.schedule_state === "scheduled" && Date.parse(lesson.start_at) > now;
}

export function hasUpcomingLessons(lessons: Lesson[], now = Date.now()): boolean {
  return lessons.some((lesson) => canManageLesson(lesson, now));
}

export function lessonParticipantName(lesson: Lesson, role: PersonaRole, assignments: readonly Assignment[]): string {
  const assignment = assignments.find((item) => item.id === lesson.assignment);
  return assignment ? assignmentDisplayName(assignment, role) : opaqueIdLabel(role === "teacher" ? lesson.learner : lesson.teacher);
}

export function LessonList({ lessons, role, policy, commercialSummaries = [], assignments }: { lessons: Lesson[]; role: PersonaRole; policy: Policy; commercialSummaries?: CommercialSummary[]; assignments: readonly Assignment[] }) {
  const [editing, setEditing] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const lifecycle = useLifecycleMutation();
  const outcome = useOutcomeMutation();
  const { notify } = useToast();
  const [pending, setPending] = useState<{ lesson: Lesson; kind: "reschedule" | "cancel"; start?: string } | null>(null);
  const summaryOf = (lesson: Lesson) => commercialSummaries.find((item) => item.assignment === lesson.assignment);
  async function reschedule(lesson: Lesson, start: string) {
    setBusy(lesson.id);
    try {
      await lifecycle.mutateAsync({ assignmentId: lesson.assignment, write: () => rescheduleCommercialLesson(role, lesson.id, { start_at: localInputToUtc(start) }) });
      notify("Lekcja przełożona.");
      setEditing(null);
    } finally { setBusy(null); }
  }
  async function cancel(lesson: Lesson) {
    setBusy(lesson.id);
    try {
      await lifecycle.mutateAsync({ assignmentId: lesson.assignment, write: () => cancelCommercialLesson(role, lesson.id) });
      notify("Lekcja odwołana.");
    } finally { setBusy(null); }
  }
  async function confirmPending() {
    if (!pending) return;
    const request = pending;
    setPending(null);
    if (request.kind === "cancel") await cancel(request.lesson);
    else if (request.start) await reschedule(request.lesson, request.start);
  }
  async function close(lesson: Lesson, value: "completed" | "learner_no_show") {
    setBusy(lesson.id);
    try {
      await outcome.mutateAsync({ assignmentId: lesson.assignment, write: () => recordLessonOutcome(lesson.id, { outcome: value }) });
      notify(`Zapisano wynik: ${outcomeCopy[value]}.`);
    } finally { setBusy(null); }
  }
  return <div className="lesson-list"><ApiFeedback error={lifecycle.error || outcome.error} />{lessons.length === 0 ? <EmptyState>Nie masz jeszcze żadnych lekcji.</EmptyState> : lessons.map((lesson) => <LessonCard key={lesson.id} lesson={lesson} role={role} policy={policy} summary={commercialSummaries.find((item) => item.assignment === lesson.assignment)} assignments={assignments} editing={editing === lesson.id} busy={busy === lesson.id} onEdit={() => setEditing(editing === lesson.id ? null : lesson.id)} onReschedule={(start) => setPending({ lesson, kind: "reschedule", start })} onCancel={() => setPending({ lesson, kind: "cancel" })} onOutcome={(value) => void close(lesson, value)} />)}{!hasUpcomingLessons(lessons) && lessons.length > 0 ? <p className="supporting-copy">Brak nadchodzących lekcji.</p> : null}<ConfirmDialog open={Boolean(pending)} title={pending?.kind === "cancel" ? "Odwołanie lekcji" : "Przełożenie lekcji"} consequence={pending ? lessonChangeNotice(pending.kind, pending.lesson, role, policy, summaryOf(pending.lesson)) : ""} confirmLabel={pending?.kind === "cancel" ? "Odwołaj lekcję" : "Przełóż lekcję"} danger={pending?.kind === "cancel"} busy={lifecycle.isPending} onConfirm={() => void confirmPending()} onCancel={() => setPending(null)} /></div>;
}

type LessonCardProps = {
  lesson: Lesson;
  role: PersonaRole;
  policy: Policy;
  summary?: CommercialSummary;
  assignments: readonly Assignment[];
  editing: boolean;
  busy: boolean;
  onEdit: () => void;
  onReschedule: (start: string) => void;
  onCancel: () => void;
  onOutcome: (value: "completed" | "learner_no_show") => void;
};

function LessonCard({ lesson, role, policy, summary, assignments, editing, busy, onEdit, onReschedule, onCancel, onOutcome }: LessonCardProps) {
  const [start, setStart] = useState(() => utcToLocalInput(lesson.start_at));
  const [selectedOutcome, setSelectedOutcome] = useState<"completed" | "learner_no_show">("completed");
  useEffect(() => setStart(utcToLocalInput(lesson.start_at)), [lesson.start_at]);
  const state = lessonCardState(lesson, role, policy, summary);
  function submit(event: FormEvent) { event.preventDefault(); onReschedule(start); }
  return <article className={`lesson-card ${state.cancelledClass}`}>
    <LessonHeading lesson={lesson} role={role} assignments={assignments} />
    <LessonStatusDetails lesson={lesson} role={role} />
    <FutureLessonActions visible={state.future} canReschedule={state.canReschedule} editing={editing} busy={busy} onEdit={onEdit} onCancel={onCancel} />
    {state.showNotice ? <p className="supporting-copy">{state.rescheduleNotice}</p> : null}
    <RescheduleForm visible={editing && state.future && state.canReschedule} lessonId={lesson.id} start={start} busy={busy} notice={state.rescheduleNotice} onStart={setStart} onSubmit={submit} />
    <OutcomeForm visible={state.awaiting} lessonId={lesson.id} value={selectedOutcome} busy={busy} onChange={setSelectedOutcome} onSubmit={onOutcome} />
  </article>;
}

function lessonCardState(lesson: Lesson, role: PersonaRole, policy: Policy, summary?: CommercialSummary) {
  const future = canManageLesson(lesson);
  const canReschedule = role === "teacher" || learnerChangeIsTimely(lesson, policy);
  return {
    future,
    canReschedule,
    showNotice: future && !canReschedule,
    cancelledClass: lesson.schedule_state === "scheduled" ? "" : "lesson-cancelled",
    rescheduleNotice: lessonChangeNotice("reschedule", lesson, role, policy, summary),
    awaiting: awaitingOutcome(lesson, role),
  };
}

function awaitingOutcome(lesson: Lesson, role: PersonaRole): boolean {
  if (role !== "teacher" || lesson.schedule_state !== "scheduled") return false;
  if (Date.parse(lesson.end_at) > Date.now()) return false;
  return !lesson.outcome || lesson.outcome === "awaiting_outcome";
}

function LessonHeading({ lesson, role, assignments }: { lesson: Lesson; role: PersonaRole; assignments: readonly Assignment[] }) {
  return <>
    <div className="lesson-heading"><div><p className="eyebrow">{planCopy[lesson.plan_type]}</p><h3>{lessonParticipantName(lesson, role, assignments)}</h3></div><span className={`status-badge status-${lesson.schedule_state}`}>{scheduleStateCopy[lesson.schedule_state]}</span></div>
    <p className="lesson-meta">{formatScheduleInstant(lesson.start_at)} – {formatScheduleInstant(lesson.end_at)} · {lesson.duration_minutes} min · {formatMoney(lesson.unit_price_minor, lesson.currency)}</p>
  </>;
}

function LessonStatusDetails({ lesson, role }: { lesson: Lesson; role: PersonaRole }) {
  return <>{lesson.outcome ? <p className="supporting-copy">Wynik: {outcomeCopy[lesson.outcome]}</p> : null}{lesson.cancellation_initiator_role ? <p className="supporting-copy">Odwołana przez {lesson.cancellation_initiator_role === role ? "Ciebie" : "drugą osobę"}.</p> : null}</>;
}

function FutureLessonActions({ visible, canReschedule, editing, busy, onEdit, onCancel }: { visible: boolean; canReschedule: boolean; editing: boolean; busy: boolean; onEdit: () => void; onCancel: () => void }) {
  if (!visible) return null;
  return <div className="lesson-actions">{canReschedule ? <button className="btn btn-ghost btn-sm" onClick={onEdit} disabled={busy}>{editing ? "Zamknij" : "Przełóż"}</button> : null}<button className="text-button danger-button" onClick={onCancel} disabled={busy}>Odwołaj</button></div>;
}

function RescheduleForm({ visible, lessonId, start, busy, notice, onStart, onSubmit }: { visible: boolean; lessonId: string; start: string; busy: boolean; notice: string; onStart: (value: string) => void; onSubmit: (event: FormEvent) => void }) {
  if (!visible) return null;
  return <form className="inline-form" onSubmit={onSubmit}><label htmlFor={`start-${lessonId}`}>Nowy termin</label><input id={`start-${lessonId}`} type="datetime-local" value={start} onChange={(event) => onStart(event.target.value)} required /><p className="supporting-copy">{notice}</p><button className="btn btn-primary btn-sm" type="submit" disabled={busy}>Zapisz termin</button></form>;
}

function OutcomeForm({ visible, lessonId, value, busy, onChange, onSubmit }: { visible: boolean; lessonId: string; value: "completed" | "learner_no_show"; busy: boolean; onChange: (value: "completed" | "learner_no_show") => void; onSubmit: (value: "completed" | "learner_no_show") => void }) {
  if (!visible) return null;
  return <form className="inline-form" onSubmit={(event) => { event.preventDefault(); onSubmit(value); }}><label htmlFor={`outcome-${lessonId}`}>Wynik lekcji</label><select id={`outcome-${lessonId}`} value={value} onChange={(event) => onChange(event.target.value as typeof value)}><option value="completed">Zrealizowana</option><option value="learner_no_show">Nieobecność ucznia</option></select><p className="supporting-copy">Domyślny wybór nie zostanie zapisany bez potwierdzenia.</p><button className="btn btn-primary btn-sm" type="submit" disabled={busy}>Zapisz wynik</button></form>;
}
