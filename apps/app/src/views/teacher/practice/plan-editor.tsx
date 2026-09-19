// Edits the practice plan of one learner in one save: each active task stays, is done, or goes to the archive, and new tasks join the plan.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { savePracticePlan, type NewTask, type PracticeTask } from "../../../api/practice";
import { ActionButton } from "../../../components/action-button";
import { ApiFeedback } from "../../../components/api-feedback";
import { Skeleton } from "../../../components/skeleton";
import { useToast } from "../../../components/toast";
import { practiceTasksQuery, usePracticeMutation } from "../../../query/practice";
import { emptyTask, NewTaskFields } from "./new-task-fields";

type Choice = "keep" | "done" | "archive";
type EditorProps = { accountId: string; assignmentId: string; lessonId?: string; onClose: () => void };

const choiceLabels: Record<Choice, string> = { keep: "Zostaje", done: "Zrobione", archive: "Do archiwum" };

export function PlanEditor(props: EditorProps) {
  const tasks = useQuery(practiceTasksQuery("teacher", props.accountId, props.assignmentId));
  if (tasks.error) return <ApiFeedback error={tasks.error} onRetry={() => void tasks.refetch()} />;
  if (!tasks.data) return <Skeleton lines={3} label="Ładowanie zadań…" />;
  return <PlanForm {...props} active={tasks.data.items} />;
}

function PlanForm({ accountId, assignmentId, lessonId, onClose, active }: EditorProps & { active: PracticeTask[] }) {
  const save = usePracticeMutation();
  const { notify } = useToast();
  const [choices, setChoices] = useState<Record<string, Choice>>(() => Object.fromEntries(active.map((task) => [task.id, "keep"])));
  const [drafts, setDrafts] = useState<NewTask[]>([emptyTask()]);
  const created = drafts.filter((draft) => draft.title.trim());
  const pick = (choice: Choice) => active.filter((task) => choices[task.id] === choice).map((task) => task.id);
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    await save.mutateAsync({ assignmentId, write: () => savePracticePlan(assignmentId, { lesson: lessonId, keep: pick("keep"), done: pick("done"), archive: pick("archive"), create: created }) });
    notify("Zapisano zadania. Uczeń widzi je od razu.");
    onClose();
  }
  return <form className="inline-form plan-form" onSubmit={(event) => void submit(event).catch(() => undefined)}>
    <h3>Zadania na tydzień</h3>
    {active.length ? <fieldset className="picker"><legend>Obecne zadania</legend>
      {active.map((task) => <div className="plan-task" key={task.id}><span>{task.title}</span>
        <select aria-label={`Zadanie ${task.title}`} value={choices[task.id]} onChange={(event) => setChoices({ ...choices, [task.id]: event.target.value as Choice })}>{(Object.keys(choiceLabels) as Choice[]).map((choice) => <option key={choice} value={choice}>{choiceLabels[choice]}</option>)}</select>
      </div>)}
    </fieldset> : null}
    {drafts.map((draft, index) => <NewTaskFields key={index} index={index} accountId={accountId} assignmentId={assignmentId} value={draft} onChange={(next) => setDrafts(drafts.map((item, position) => (position === index ? next : item)))} />)}
    <ActionButton type="button" onClick={() => setDrafts([...drafts, emptyTask()])}>Dodaj kolejne zadanie</ActionButton>
    <ApiFeedback error={save.error} />
    <div className="row-actions">
      <ActionButton variant="text" type="button" onClick={onClose}>Anuluj</ActionButton>
      <ActionButton variant="primary" type="submit" busy={save.isPending}>Zapisz zadania</ActionButton>
    </div>
  </form>;
}
