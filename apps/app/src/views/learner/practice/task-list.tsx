// Shows the active practice tasks with details, a suggested time, and the linked piece and material.

import { Link } from "@tanstack/react-router";

import type { PracticeTask } from "../../../api/practice";
import { EmptyState } from "../../../components/empty-state";

export function TaskList({ tasks, assignmentId }: { tasks: PracticeTask[]; assignmentId: string }) {
  if (tasks.length === 0) return <EmptyState>Nauczyciel dodaje zadania po lekcji. Ćwiczenie możesz zapisać także bez zadań.</EmptyState>;
  return <ol className="task-list">{tasks.map((task) => <li key={task.id} className="task-card">
    <h3>{task.title}</h3>
    {task.details ? <p className="task-details">{task.details}</p> : null}
    <TaskMeta task={task} assignmentId={assignmentId} />
  </li>)}</ol>;
}

function TaskMeta({ task, assignmentId }: { task: PracticeTask; assignmentId: string }) {
  if (!task.suggested_minutes && !task.piece && !task.material) return null;
  return <p className="lesson-meta">
    {task.suggested_minutes ? <span>ok. {task.suggested_minutes} min dziennie</span> : null}
    {task.piece ? <Link to="/learners/pieces" search={{ a: assignmentId }}>Utwór: {task.piece.title}</Link> : null}
    {task.material ? <span>Materiał: {task.material.title}</span> : null}
  </p>;
}
