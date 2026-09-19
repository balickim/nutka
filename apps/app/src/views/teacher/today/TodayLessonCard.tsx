// Records the outcome of one of today's lessons in a single step, without leaving the Today screen.

import { useState } from "react";

import { recordLessonOutcome } from "../../../api/commercial";
import { outcomeCopy } from "../../../api/copy";
import type { Lesson, Policy } from "../../../api/contracts";
import { ActionButton } from "../../../components/ActionButton";
import { ApiFeedback } from "../../../components/ApiFeedback";
import { useToast } from "../../../components/Toast";
import { useOutcomeMutation } from "../../../query/commercial";
import { formatScheduleInstant } from "../../../time/schedule";

type Outcome = "completed" | "learner_no_show";

export function TodayLessonCard({ lesson, learner, policy }: { lesson: Lesson; learner?: string; policy: Policy }) {
  const mutation = useOutcomeMutation();
  const { notify } = useToast();
  const [busy, setBusy] = useState<Outcome | null>(null);
  const ended = Date.parse(lesson.end_at) <= Date.now();
  const recorded = Boolean(lesson.outcome && lesson.outcome !== "awaiting_outcome");
  async function record(outcome: Outcome) {
    setBusy(outcome);
    try {
      await mutation.mutateAsync({ assignmentId: lesson.assignment, write: () => recordLessonOutcome(lesson.id, { outcome }) });
      notify(`Zapisano wynik: ${outcomeCopy[outcome]}.`);
    } finally { setBusy(null); }
  }
  return <article className="lesson-card today-lesson">
    <div className="lesson-heading"><div><h3>{learner ?? "Uczeń"}</h3><p className="lesson-meta">{formatScheduleInstant(lesson.start_at)} · {policy.lesson_duration_minutes} min</p></div>
      {recorded ? <span className="status-badge status-settled">{outcomeCopy[lesson.outcome!]}</span> : null}</div>
    <ApiFeedback error={mutation.error} />
    {ended && !recorded ? <div className="lesson-actions">
      <ActionButton busy={busy === "completed"} onClick={() => void record("completed")}>Odbyta</ActionButton>
      <ActionButton busy={busy === "learner_no_show"} onClick={() => void record("learner_no_show")}>Nieobecność</ActionButton>
    </div> : null}
  </article>;
}
