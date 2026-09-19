// Shows the practice of one learner to the teacher: the summary since the last lesson, the active tasks with a plan editor, and the recorded sessions.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { ActionButton } from "../../../components/action-button";
import { ApiFeedback } from "../../../components/api-feedback";
import { EmptyState } from "../../../components/empty-state";
import type { PracticeTask } from "../../../api/practice";
import { SessionList } from "../../../components/practice-session-list";
import { Skeleton } from "../../../components/skeleton";
import { practiceSessionsQuery, practiceSummaryQuery, practiceTasksQuery } from "../../../query/practice";
import { PlanEditor } from "../practice/plan-editor";
import { PracticeSummaryBlock } from "../practice/practice-summary";

export function PracticeTab({ accountId, assignmentId }: { accountId: string; assignmentId: string }) {
  const tasks = useQuery(practiceTasksQuery("teacher", accountId, assignmentId));
  const summary = useQuery(practiceSummaryQuery("teacher", accountId, assignmentId));
  const sessions = useQuery(practiceSessionsQuery("teacher", accountId, assignmentId));
  const failure = tasks.error ?? summary.error ?? sessions.error;
  if (failure) return <ApiFeedback error={failure} onRetry={() => { void tasks.refetch(); void summary.refetch(); void sessions.refetch(); }} />;
  if (!tasks.data || !summary.data || !sessions.data) return <Skeleton lines={5} label="Ładowanie ćwiczeń…" />;
  return <>
    <PracticeSummaryBlock summary={summary.data} />
    <h3>Zadania na tydzień</h3>
    <TaskPlan accountId={accountId} assignmentId={assignmentId} tasks={tasks.data.items} />
    <h3>Dziennik ćwiczeń</h3>
    <SessionList assignmentId={assignmentId} sessions={sessions.data.items} empty="Uczeń nie zapisał jeszcze ćwiczeń." />
  </>;
}

function TaskPlan({ accountId, assignmentId, tasks }: { accountId: string; assignmentId: string; tasks: PracticeTask[] }) {
  const [editing, setEditing] = useState(false);
  if (editing) return <PlanEditor accountId={accountId} assignmentId={assignmentId} onClose={() => setEditing(false)} />;
  return <>
    {tasks.length ? <ol className="task-titles">{tasks.map((task) => <li key={task.id}>{task.title}{task.suggested_minutes ? <span className="lesson-meta"> · ok. {task.suggested_minutes} min</span> : null}</li>)}</ol> : <EmptyState>Ten uczeń nie ma zadań.</EmptyState>}
    <ActionButton onClick={() => setEditing(true)}>Zmień zadania</ActionButton>
  </>;
}
