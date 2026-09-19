// Lists the learner's past lessons with their teacher notes and lets the teacher add, edit, or delete a note.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { outcomeCopy } from "../../../api/copy";
import type { Lesson } from "../../../api/contracts";
import { deleteLessonNote, type LessonNote } from "../../../api/lesson-notes";
import { ApiFeedback } from "../../../components/api-feedback";
import { ConfirmDialog } from "../../../components/confirm-dialog";
import { EmptyState } from "../../../components/empty-state";
import { Skeleton } from "../../../components/skeleton";
import { useToast } from "../../../components/toast";
import { lessonNotesQuery, useNoteMutation } from "../../../query/lesson-notes";
import { formatScheduleInstant } from "../../../time/schedule";
import { NoteEditor } from "../lesson-note/note-editor";

type TabProps = { accountId: string; assignmentId: string; pastLessons: Lesson[] };

export function NotesTab({ accountId, assignmentId, pastLessons }: TabProps) {
  const notes = useQuery(lessonNotesQuery("teacher", accountId, assignmentId));
  const remove = useNoteMutation();
  const { notify } = useToast();
  const [editing, setEditing] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<Lesson | null>(null);
  if (notes.error) return <ApiFeedback error={notes.error} onRetry={() => void notes.refetch()} />;
  if (!notes.data) return <Skeleton lines={4} label="Ładowanie notatek…" />;
  if (pastLessons.length === 0) return <EmptyState>Ten uczeń nie miał jeszcze lekcji.</EmptyState>;
  const byLesson = new Map(notes.data.items.map((note) => [note.lesson, note]));
  async function confirmDelete() {
    if (!deleting) return;
    await remove.mutateAsync({ assignmentId, write: () => deleteLessonNote(deleting.id) });
    notify("Usunięto notatkę.");
    setDeleting(null);
  }
  return <div className="note-lessons">
    {pastLessons.map((lesson) => <article className="lesson-card" key={lesson.id}>
      <div className="lesson-heading"><h3>{formatScheduleInstant(lesson.start_at)}</h3>{lesson.outcome && lesson.outcome !== "awaiting_outcome" ? <span className="status-badge status-settled">{outcomeCopy[lesson.outcome]}</span> : null}</div>
      {editing === lesson.id
        ? <NoteEditor accountId={accountId} assignmentId={assignmentId} lessonId={lesson.id} onClose={() => setEditing(null)} />
        : <NoteSummary note={byLesson.get(lesson.id)} onEdit={() => setEditing(lesson.id)} onDelete={() => setDeleting(lesson)} />}
    </article>)}
    <ConfirmDialog open={deleting !== null} title="Usunięcie notatki" consequence={`Notatka z lekcji ${deleting ? formatScheduleInstant(deleting.start_at) : ""} zniknie także z panelu ucznia.`} confirmLabel="Usuń notatkę" danger busy={remove.isPending} onConfirm={() => void confirmDelete().catch(() => undefined)} onCancel={() => setDeleting(null)}>
      <ApiFeedback error={remove.error} />
    </ConfirmDialog>
  </div>;
}

function NoteSummary({ note, onEdit, onDelete }: { note?: LessonNote; onEdit: () => void; onDelete: () => void }) {
  if (!note) return <div className="row-actions"><p className="supporting-copy">Brak notatki.</p><button className="btn btn-ghost btn-sm" onClick={onEdit}>Dodaj notatkę</button></div>;
  return <>
    {/* The backend stores only sanitized HTML, so rendering it cannot run scripts. */}
    <div className="material-body" dangerouslySetInnerHTML={{ __html: note.body }} />
    {note.materials.length ? <p className="supporting-copy">Materiały: {note.materials.map((item) => item.title).join(", ")}</p> : null}
    <div className="row-actions"><button className="btn btn-ghost btn-sm" onClick={onEdit}>Edytuj</button><button className="text-button danger-button" onClick={onDelete}>Usuń</button></div>
  </>;
}
