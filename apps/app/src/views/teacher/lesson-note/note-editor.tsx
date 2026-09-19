// Edits the teacher note of one lesson: formatted text and links to the learner's materials. The learner sees the note right after the save.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { saveLessonNote, type LessonNote } from "../../../api/lesson-notes";
import { ActionButton } from "../../../components/action-button";
import { ApiFeedback } from "../../../components/api-feedback";
import { RichTextEditor } from "../../../components/rich-text-editor";
import { Skeleton } from "../../../components/skeleton";
import { useToast } from "../../../components/toast";
import { lessonNotesQuery, useNoteMutation } from "../../../query/lesson-notes";
import { materialsQuery } from "../../../query/materials";

type EditorProps = { accountId: string; assignmentId: string; lessonId: string; onClose: () => void };

// Loads the existing note first, so the editor opens with the saved text.
export function NoteEditor(props: EditorProps) {
  const notes = useQuery(lessonNotesQuery("teacher", props.accountId, props.assignmentId));
  if (notes.error) return <ApiFeedback error={notes.error} onRetry={() => void notes.refetch()} />;
  if (!notes.data) return <Skeleton lines={3} label="Ładowanie notatki…" />;
  return <NoteForm {...props} existing={notes.data.items.find((note) => note.lesson === props.lessonId)} />;
}

function NoteForm({ accountId, assignmentId, lessonId, onClose, existing }: EditorProps & { existing?: LessonNote }) {
  const mutation = useNoteMutation();
  const { notify } = useToast();
  const [body, setBody] = useState(existing?.body ?? "");
  const [linked, setLinked] = useState<string[]>(existing?.materials.map((item) => item.id) ?? []);
  async function save(event: React.FormEvent) {
    event.preventDefault();
    await mutation.mutateAsync({ assignmentId, write: () => saveLessonNote(lessonId, { body, materials: linked }) });
    notify("Zapisano notatkę. Uczeń widzi ją od razu.");
    onClose();
  }
  return <form className="inline-form note-form" onSubmit={(event) => void save(event).catch(() => undefined)}>
    <label>Notatka po lekcji</label>
    <p className="supporting-copy">Co wyszło, nad czym pracujemy, co przygotować na następny raz. Dwa lub trzy zdania wystarczą.</p>
    <RichTextEditor label="Notatka po lekcji" initialHtml={existing?.body} onChange={setBody} />
    <MaterialPicker accountId={accountId} assignmentId={assignmentId} linked={linked} onChange={setLinked} />
    <ApiFeedback error={mutation.error} />
    <div className="row-actions">
      <ActionButton variant="text" type="button" onClick={onClose}>Anuluj</ActionButton>
      <ActionButton variant="primary" type="submit" busy={mutation.isPending} disabled={!hasText(body)}>Zapisz notatkę</ActionButton>
    </div>
  </form>;
}

function MaterialPicker({ accountId, assignmentId, linked, onChange }: { accountId: string; assignmentId: string; linked: string[]; onChange: (ids: string[]) => void }) {
  const materials = useQuery(materialsQuery("teacher", accountId, assignmentId));
  const toggle = (id: string) => onChange(linked.includes(id) ? linked.filter((item) => item !== id) : [...linked, id]);
  if (!materials.data?.items.length) return null;
  return <fieldset className="picker"><legend>Materiały do tej lekcji</legend>
    {materials.data.items.map((material) => <label key={material.id} className="check-row"><input type="checkbox" checked={linked.includes(material.id)} onChange={() => toggle(material.id)} /> {material.title}</label>)}
  </fieldset>;
}

function hasText(html: string): boolean {
  return html.replace(/<[^>]*>|&nbsp;/g, "").trim() !== "";
}
