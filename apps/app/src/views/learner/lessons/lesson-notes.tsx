// Shows teacher notes to the learner: the latest note on Start and every note on the lessons screen, with lesson dates and linked materials.

import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import type { LessonNote } from "../../../api/lesson-notes";
import { ApiFeedback } from "../../../components/api-feedback";
import { EmptyState } from "../../../components/empty-state";
import { Skeleton } from "../../../components/skeleton";
import { lessonNotesQuery } from "../../../query/lesson-notes";
import { formatScheduleDate } from "../../../time/schedule";

type Scope = { accountId: string; assignmentId: string };

export function LatestNote({ accountId, assignmentId }: Scope) {
  const notes = useQuery(lessonNotesQuery("learner", accountId, assignmentId));
  const latest = notes.data?.items[0];
  if (!latest) return null;
  return <section className="panel-section"><h2>Po ostatniej lekcji</h2><NoteCard note={latest} assignmentId={assignmentId} />
    <Link className="text-button" to="/learners/lessons" search={{ a: assignmentId }} hash="notatki">Wszystkie notatki</Link>
  </section>;
}

export function NoteList({ accountId, assignmentId }: Scope) {
  const notes = useQuery(lessonNotesQuery("learner", accountId, assignmentId));
  return <section className="panel-section" id="notatki"><h2>Notatki z lekcji</h2>
    {notes.error ? <ApiFeedback error={notes.error} onRetry={() => void notes.refetch()} /> : null}
    {notes.isPending ? <Skeleton lines={3} label="Ładowanie notatek…" /> : notes.data?.items.length
      ? <div className="note-list">{notes.data.items.map((note) => <NoteCard key={note.id} note={note} assignmentId={assignmentId} />)}</div>
      : <EmptyState>Po lekcjach pojawią się tu notatki od nauczyciela.</EmptyState>}
  </section>;
}

function NoteCard({ note, assignmentId }: { note: LessonNote; assignmentId: string }) {
  return <article className="material-card note-card">
    <p className="lesson-meta">Lekcja {formatScheduleDate(note.lesson_start_at)}</p>
    {/* The backend stores only sanitized HTML, so rendering it cannot run scripts. */}
    <div className="material-body" dangerouslySetInnerHTML={{ __html: note.body }} />
    {note.materials.length ? <p className="supporting-copy">Materiały: {note.materials.map((item, index) => <span key={item.id}>{index ? ", " : ""}<Link to="/learners/pieces" search={{ a: assignmentId }}>{item.title}</Link></span>)}</p> : null}
  </article>;
}
