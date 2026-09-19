// Records one practice session: the day, the practiced tasks, and optional minutes and a comment. The button keeps the form closed until the learner needs it.

import { useState } from "react";

import { createPracticeSession, practiceLimits, type PracticeTask } from "../../../api/practice";
import { ActionButton } from "../../../components/action-button";
import { ApiFeedback } from "../../../components/api-feedback";
import { useToast } from "../../../components/toast";
import { usePracticeMutation } from "../../../query/practice";
import { dayChoices } from "./days";

type FormProps = { assignmentId: string; tasks: PracticeTask[]; today: string };

export function LogPractice(props: FormProps) {
  const [open, setOpen] = useState(false);
  if (!open) return <button className="btn btn-primary practice-button" onClick={() => setOpen(true)}>Zapisz ćwiczenie</button>;
  return <SessionForm {...props} onClose={() => setOpen(false)} />;
}

function SessionForm({ assignmentId, tasks, today, onClose }: FormProps & { onClose: () => void }) {
  const save = usePracticeMutation();
  const { notify } = useToast();
  const [day, setDay] = useState(today);
  const [picked, setPicked] = useState<string[]>(tasks.map((task) => task.id));
  const [minutes, setMinutes] = useState("");
  const [comment, setComment] = useState("");
  const toggle = (id: string) => setPicked((current) => (current.includes(id) ? current.filter((item) => item !== id) : [...current, id]));
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    await save.mutateAsync({ assignmentId, write: () => createPracticeSession(assignmentId, { practiced_on: day, minutes: minutes ? Number(minutes) : null, tasks: picked, comment }) });
    notify("Zapisano ćwiczenie. Nauczyciel zobaczy je przed lekcją.");
    onClose();
  }
  return <form className="inline-form session-form" onSubmit={(event) => void submit(event).catch(() => undefined)}>
    <label htmlFor="practice-day">Kiedy</label>
    <select id="practice-day" value={day} onChange={(event) => setDay(event.target.value)}>{dayChoices(today, practiceLimits.backfillDays).map((choice) => <option key={choice.value} value={choice.value}>{choice.label}</option>)}</select>
    {tasks.length ? <fieldset className="picker"><legend>Które zadania</legend>
      {tasks.map((task) => <label key={task.id} className="check-row"><input type="checkbox" checked={picked.includes(task.id)} onChange={() => toggle(task.id)} /> {task.title}</label>)}
    </fieldset> : null}
    <label htmlFor="practice-minutes">Ile minut (opcjonalnie)</label>
    <input id="practice-minutes" type="number" inputMode="numeric" min={1} max={practiceLimits.sessionMinutesMax} value={minutes} onChange={(event) => setMinutes(event.target.value)} />
    <label htmlFor="practice-comment">Komentarz dla nauczyciela (opcjonalnie)</label>
    <textarea id="practice-comment" rows={3} maxLength={practiceLimits.commentMax} placeholder="Na przykład: takt 5 jeszcze nie wychodzi." value={comment} onChange={(event) => setComment(event.target.value)} />
    <ApiFeedback error={save.error} />
    <div className="row-actions">
      <ActionButton variant="text" type="button" onClick={onClose}>Anuluj</ActionButton>
      <ActionButton variant="primary" type="submit" busy={save.isPending}>Zapisz</ActionButton>
    </div>
  </form>;
}
