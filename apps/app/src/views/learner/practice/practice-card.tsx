// Shows the practice plan on Start with the same session button as the practice screen. It hides itself without tasks and sessions.

import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { practiceSummaryQuery, practiceTasksQuery } from "../../../query/practice";
import { learnerPracticeLine } from "./days";
import { LogPractice } from "./session-form";

export function PracticeCard({ accountId, assignmentId }: { accountId: string; assignmentId: string }) {
  const tasks = useQuery(practiceTasksQuery("learner", accountId, assignmentId));
  const summary = useQuery(practiceSummaryQuery("learner", accountId, assignmentId));
  if (!tasks.data || !summary.data) return null;
  if (tasks.data.items.length === 0 && summary.data.sessions === 0) return null;
  return <section className="panel-section practice-card">
    <h2>Na ten tydzień</h2>
    {tasks.data.items.length ? <ul className="task-titles">{tasks.data.items.map((task) => <li key={task.id}>{task.title}{task.suggested_minutes ? <span className="lesson-meta"> · ok. {task.suggested_minutes} min</span> : null}</li>)}</ul> : null}
    <p className="supporting-copy">{learnerPracticeLine(summary.data)}</p>
    <div className="row-actions"><LogPractice assignmentId={assignmentId} tasks={tasks.data.items} today={summary.data.today} /><Link className="text-button" to="/learners/practice" search={{ a: assignmentId }}>Szczegóły zadań</Link></div>
  </section>;
}
