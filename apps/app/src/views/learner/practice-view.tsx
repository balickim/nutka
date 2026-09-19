// Shows the practice plan of the selected assignment, the session form, the last four weeks, and the recorded sessions.

import { useQuery } from "@tanstack/react-query";

import { ApiFeedback } from "../../components/api-feedback";
import { Skeleton } from "../../components/skeleton";
import { practiceSessionsQuery, practiceSummaryQuery, practiceTasksQuery } from "../../query/practice";
import { LearnerShell } from "./learner-shell";
import { learnerPracticeLine } from "./practice/days";
import { PracticeGrid } from "./practice/practice-grid";
import { LogPractice } from "./practice/session-form";
import { SessionList } from "../../components/practice-session-list";
import { TaskList } from "./practice/task-list";

export function PracticeView() {
  return <LearnerShell lede="Co ćwiczyć między lekcjami i zapis Twoich ćwiczeń.">{(context) => <Practice accountId={context.accountId} assignmentId={context.assignment.id} />}</LearnerShell>;
}

function Practice({ accountId, assignmentId }: { accountId: string; assignmentId: string }) {
  const tasks = useQuery(practiceTasksQuery("learner", accountId, assignmentId));
  const summary = useQuery(practiceSummaryQuery("learner", accountId, assignmentId));
  const sessions = useQuery(practiceSessionsQuery("learner", accountId, assignmentId));
  const failure = tasks.error ?? summary.error ?? sessions.error;
  if (failure) return <section className="panel-section"><ApiFeedback error={failure} onRetry={() => { void tasks.refetch(); void summary.refetch(); void sessions.refetch(); }} /></section>;
  if (!tasks.data || !summary.data || !sessions.data) return <section className="panel-section"><Skeleton lines={5} label="Ładowanie ćwiczeń…" /></section>;
  return <>
    <section className="panel-section">
      <h2>Na ten tydzień</h2>
      <TaskList tasks={tasks.data.items} assignmentId={assignmentId} />
      <LogPractice assignmentId={assignmentId} tasks={tasks.data.items} today={summary.data.today} />
    </section>
    <section className="panel-section">
      <h2>Twoje ćwiczenia</h2>
      <p className="supporting-copy">{learnerPracticeLine(summary.data)}</p>
      <PracticeGrid today={summary.data.today} days={summary.data.recent_days} />
      <h3>Zapisane wpisy</h3>
      <SessionList assignmentId={assignmentId} sessions={sessions.data.items} empty="Nie masz jeszcze zapisanych ćwiczeń." />
    </section>
  </>;
}
