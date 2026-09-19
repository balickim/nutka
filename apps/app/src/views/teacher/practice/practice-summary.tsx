// Shows what the learner practiced since the previous lesson: days, minutes, the most practiced tasks, and the learner comments.

import type { PracticeSummary } from "../../../api/practice";
import { teacherPracticeLine } from "../../learner/practice/days";
import { formatLocalDate } from "../../../time/schedule";

export function PracticeSummaryBlock({ summary }: { summary: PracticeSummary }) {
  return <div className="practice-summary">
    <p><strong>{teacherPracticeLine(summary)}</strong></p>
    {summary.tasks.length ? <p className="lesson-meta">{summary.tasks.map((task) => `${task.title} (${task.count}×)`).join(", ")}</p> : null}
    {summary.comments.length ? <ul className="session-comments">{summary.comments.slice(0, 3).map((comment) => <li key={`${comment.practiced_on}-${comment.comment}`}><span className="lesson-meta">{formatLocalDate(comment.practiced_on)}</span> „{comment.comment}”</li>)}</ul> : null}
  </div>;
}
