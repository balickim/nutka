// Lets the teacher add rich-text, image, and PDF materials for one learner and delete them. Only this learner can read them.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { createMaterial, deleteMaterial, materialLimits, type Material } from "../../../api/materials";
import { ActionButton } from "../../../components/action-button";
import { ApiFeedback } from "../../../components/api-feedback";
import { ConfirmDialog } from "../../../components/confirm-dialog";
import { MaterialList } from "../../../components/material-list";
import { RichTextEditor } from "../../../components/rich-text-editor";
import { Skeleton } from "../../../components/skeleton";
import { useToast } from "../../../components/toast";
import { materialsQuery, useMaterialMutation } from "../../../query/materials";

export function MaterialsTab({ accountId, assignmentId }: { accountId: string; assignmentId: string }) {
  const materials = useQuery(materialsQuery("teacher", accountId, assignmentId));
  const remove = useMaterialMutation();
  const { notify } = useToast();
  const [pending, setPending] = useState<Material | null>(null);
  async function confirmDelete() {
    if (!pending) return;
    await remove.mutateAsync({ assignmentId, write: () => deleteMaterial(pending.id) });
    notify(`Usunięto materiał „${pending.title}”.`);
    setPending(null);
  }
  return <>
    <MaterialForm assignmentId={assignmentId} />
    <h3>Materiały ucznia</h3>
    {materials.error ? <ApiFeedback error={materials.error} onRetry={() => void materials.refetch()} /> : null}
    {materials.isPending ? <Skeleton lines={4} label="Ładowanie materiałów…" /> : <MaterialList items={materials.data?.items ?? []} empty="Ten uczeń nie ma jeszcze materiałów." onDelete={setPending} />}
    <ConfirmDialog open={pending !== null} title="Usunięcie materiału" consequence={`Materiał „${pending?.title ?? ""}” i jego załączniki (${pending?.attachments.length ?? 0}) znikną także z panelu ucznia.`} confirmLabel="Usuń materiał" danger busy={remove.isPending} onConfirm={() => void confirmDelete().catch(() => undefined)} onCancel={() => setPending(null)}>
      <ApiFeedback error={remove.error} />
    </ConfirmDialog>
  </>;
}

function MaterialForm({ assignmentId }: { assignmentId: string }) {
  const save = useMaterialMutation();
  const { notify } = useToast();
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [version, setVersion] = useState(0);
  const fileError = filesProblem(files);
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    await save.mutateAsync({ assignmentId, write: () => createMaterial(assignmentId, { title, body, files }) });
    notify(`Dodano materiał „${title.trim()}”.`);
    setTitle(""); setBody(""); setFiles([]); setVersion((value) => value + 1);
  }
  return <form className="inline-form material-form" onSubmit={(event) => void submit(event).catch(() => undefined)}>
    <h3>Nowy materiał</h3>
    <label htmlFor="material-title">Tytuł</label>
    <input id="material-title" value={title} maxLength={200} required onChange={(event) => setTitle(event.target.value)} />
    <label>Treść</label>
    <RichTextEditor key={`body-${version}`} label="Treść materiału" onChange={setBody} />
    <label htmlFor="material-files">Zdjęcia i PDF</label>
    <input key={`files-${version}`} id="material-files" type="file" multiple accept={materialLimits.accept} aria-invalid={Boolean(fileError)} onChange={(event) => setFiles(Array.from(event.target.files ?? []))} />
    {fileError ? <p className="field-error">{fileError}</p> : <p className="supporting-copy">Do {materialLimits.maxFiles} plików, każdy do 10 MB. Uczeń zobaczy materiał od razu.</p>}
    <ApiFeedback error={save.error} />
    <ActionButton variant="primary" type="submit" busy={save.isPending} disabled={!title.trim() || Boolean(fileError)}>Dodaj materiał</ActionButton>
  </form>;
}

function filesProblem(files: File[]): string | null {
  if (files.length > materialLimits.maxFiles) return `Wybierz najwyżej ${materialLimits.maxFiles} plików.`;
  const allowed = materialLimits.accept.split(",");
  if (files.some((file) => !allowed.includes(file.type))) return "Dozwolone są tylko zdjęcia (JPG, PNG, WebP, GIF) i pliki PDF.";
  if (files.some((file) => file.size > materialLimits.maxFileBytes)) return "Każdy plik może mieć najwyżej 10 MB.";
  return null;
}
