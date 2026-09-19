// Lets the teacher add rich-text, image, and PDF materials for one learner, link them to a piece, and delete them. Only this learner can read them.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { createMaterial, deleteMaterial, materialLimits, pinMaterial, type Material } from "../../../api/materials";
import type { Piece } from "../../../api/pieces";
import { ActionButton } from "../../../components/action-button";
import { ApiFeedback } from "../../../components/api-feedback";
import { ConfirmDialog } from "../../../components/confirm-dialog";
import { MaterialList } from "../../../components/material-list";
import { RichTextEditor } from "../../../components/rich-text-editor";
import { Skeleton } from "../../../components/skeleton";
import { useToast } from "../../../components/toast";
import { materialsQuery, useMaterialMutation } from "../../../query/materials";
import { piecesQuery } from "../../../query/pieces";

export function MaterialsTab({ accountId, assignmentId }: { accountId: string; assignmentId: string }) {
  const materials = useQuery(materialsQuery("teacher", accountId, assignmentId));
  const pieces = useQuery(piecesQuery("teacher", accountId, assignmentId)).data?.items ?? [];
  const titles = new Map(pieces.map((piece) => [piece.id, piece.title]));
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
    <MaterialForm assignmentId={assignmentId} pieces={pieces} />
    <h3>Materiały ucznia</h3>
    {materials.error ? <ApiFeedback error={materials.error} onRetry={() => void materials.refetch()} /> : null}
    {materials.isPending ? <Skeleton lines={4} label="Ładowanie materiałów…" /> : <MaterialList items={materials.data?.items ?? []} empty="Ten uczeń nie ma jeszcze materiałów." caption={(material) => (material.piece ? titles.get(material.piece) ?? null : null)} actions={(material) => <><PinSelect assignmentId={assignmentId} material={material} pieces={pieces} /><button className="text-button danger-button" onClick={() => setPending(material)}>Usuń</button></>} />}
    <ConfirmDialog open={pending !== null} title="Usunięcie materiału" consequence={`Materiał „${pending?.title ?? ""}” i jego załączniki (${pending?.attachments.length ?? 0}) znikną także z panelu ucznia.`} confirmLabel="Usuń materiał" danger busy={remove.isPending} onConfirm={() => void confirmDelete().catch(() => undefined)} onCancel={() => setPending(null)}>
      <ApiFeedback error={remove.error} />
    </ConfirmDialog>
  </>;
}

function MaterialForm({ assignmentId, pieces }: { assignmentId: string; pieces: Piece[] }) {
  const save = useMaterialMutation();
  const { notify } = useToast();
  const [title, setTitle] = useState("");
  const [piece, setPiece] = useState("");
  const [body, setBody] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [version, setVersion] = useState(0);
  const fileError = filesProblem(files);
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    await save.mutateAsync({ assignmentId, write: () => createMaterial(assignmentId, { title, body, files, piece }) });
    notify(`Dodano materiał „${title.trim()}”.`);
    setTitle(""); setBody(""); setFiles([]); setPiece(""); setVersion((value) => value + 1);
  }
  return <form className="inline-form material-form" onSubmit={(event) => void submit(event).catch(() => undefined)}>
    <h3>Nowy materiał</h3>
    <label htmlFor="material-title">Tytuł</label>
    <input id="material-title" value={title} maxLength={200} required onChange={(event) => setTitle(event.target.value)} />
    {pieces.length ? <><label htmlFor="material-piece">Utwór</label><PieceOptions id="material-piece" value={piece} pieces={pieces} onChange={setPiece} /></> : null}
    <label>Treść</label>
    <RichTextEditor key={`body-${version}`} label="Treść materiału" onChange={setBody} />
    <label htmlFor="material-files">Zdjęcia i PDF</label>
    <input key={`files-${version}`} id="material-files" type="file" multiple accept={materialLimits.accept} aria-invalid={Boolean(fileError)} onChange={(event) => setFiles(Array.from(event.target.files ?? []))} />
    {fileError ? <p className="field-error">{fileError}</p> : <p className="supporting-copy">Do {materialLimits.maxFiles} plików, każdy do 10 MB. Uczeń zobaczy materiał od razu.</p>}
    <ApiFeedback error={save.error} />
    <ActionButton variant="primary" type="submit" busy={save.isPending} disabled={!title.trim() || Boolean(fileError)}>Dodaj materiał</ActionButton>
  </form>;
}

function PieceOptions({ id, value, pieces, label, disabled, onChange }: { id?: string; value: string; pieces: Piece[]; label?: string; disabled?: boolean; onChange: (piece: string) => void }) {
  return <select id={id} aria-label={label} value={value} disabled={disabled} onChange={(event) => onChange(event.target.value)}>
    <option value="">Bez utworu</option>
    {pieces.map((piece) => <option key={piece.id} value={piece.id}>{piece.title}</option>)}
  </select>;
}

// A change pins the material at once. The material keeps its creation date, so its version number follows the creation order within the piece.
function PinSelect({ assignmentId, material, pieces }: { assignmentId: string; material: Material; pieces: Piece[] }) {
  const pin = useMaterialMutation();
  const { notify } = useToast();
  if (pieces.length === 0) return null;
  async function change(piece: string) {
    await pin.mutateAsync({ assignmentId, write: () => pinMaterial(material.id, piece) });
    notify(piece ? `Przypięto „${material.title}” do utworu.` : `Odpięto „${material.title}” od utworu.`);
  }
  return <><PieceOptions value={material.piece ?? ""} pieces={pieces} label={`Utwór materiału ${material.title}`} disabled={pin.isPending} onChange={(piece) => void change(piece).catch(() => undefined)} /><ApiFeedback error={pin.error} /></>;
}

function filesProblem(files: File[]): string | null {
  if (files.length > materialLimits.maxFiles) return `Wybierz najwyżej ${materialLimits.maxFiles} plików.`;
  const allowed = materialLimits.accept.split(",");
  if (files.some((file) => !allowed.includes(file.type))) return "Dozwolone są tylko zdjęcia (JPG, PNG, WebP, GIF) i pliki PDF.";
  if (files.some((file) => file.size > materialLimits.maxFileBytes)) return "Każdy plik może mieć najwyżej 10 MB.";
  return null;
}
