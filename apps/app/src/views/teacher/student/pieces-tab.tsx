// Lets the teacher manage the pieces of one learner: learner wishes first, a create form, a status change per piece, and delete.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { arrangementCount, pieceStatusCopy } from "../../../api/copy";
import { deletePiece, updatePiece, type Piece, type PieceStatus } from "../../../api/pieces";
import { ActionButton } from "../../../components/action-button";
import { ApiFeedback } from "../../../components/api-feedback";
import { ConfirmDialog } from "../../../components/confirm-dialog";
import { EmptyState } from "../../../components/empty-state";
import { Skeleton } from "../../../components/skeleton";
import { useToast } from "../../../components/toast";
import { piecesQuery, usePieceMutation } from "../../../query/pieces";
import { PieceForm } from "./pieces/piece-form";

type Props = { accountId: string; assignmentId: string };

const statusOptions: PieceStatus[] = ["learning", "playing", "repertoire", "wish"];

export function PiecesTab({ accountId, assignmentId }: Props) {
  const pieces = useQuery(piecesQuery("teacher", accountId, assignmentId));
  const write = usePieceMutation();
  const { notify } = useToast();
  const [deleting, setDeleting] = useState<Piece | null>(null);
  async function setStatus(piece: Piece, status: PieceStatus) {
    await write.mutateAsync({ assignmentId, write: () => updatePiece(piece.id, { status }) });
    notify(`„${piece.title}”: ${pieceStatusCopy.teacher[status]}.`);
  }
  async function confirmDelete() {
    if (!deleting) return;
    await write.mutateAsync({ assignmentId, write: () => deletePiece("teacher", deleting.id) });
    notify(`Usunięto utwór „${deleting.title}”.`);
    setDeleting(null);
  }
  if (pieces.error) return <ApiFeedback error={pieces.error} onRetry={() => void pieces.refetch()} />;
  if (!pieces.data) return <Skeleton lines={4} label="Ładowanie utworów…" />;
  const wishes = pieces.data.items.filter((piece) => piece.status === "wish");
  const others = pieces.data.items.filter((piece) => piece.status !== "wish");
  const onStatus = (piece: Piece, status: PieceStatus) => void setStatus(piece, status).catch(() => undefined);
  return <>
    {wishes.length ? <Wishes wishes={wishes} busy={write.isPending} onStart={(piece) => onStatus(piece, "learning")} onDelete={setDeleting} /> : null}
    <ApiFeedback error={deleting ? null : write.error} />
    <h3 className="pieces-heading">Utwory ucznia</h3>
    {others.length === 0 ? <EmptyState>Ten uczeń nie ma jeszcze utworów.</EmptyState> : <div className="piece-rows">{others.map((piece) => <PieceRow key={piece.id} piece={piece} disabled={write.isPending} onStatus={(status) => onStatus(piece, status)} onDelete={() => setDeleting(piece)} />)}</div>}
    <PieceForm assignmentId={assignmentId} />
    <ConfirmDialog open={deleting !== null} title="Usunięcie utworu" consequence={`Utwór „${deleting?.title ?? ""}” zniknie także z panelu ucznia. Jego opracowania (${deleting?.material_count ?? 0}) zostaną w materiałach bez utworu.`} confirmLabel="Usuń utwór" danger busy={write.isPending} onConfirm={() => void confirmDelete().catch(() => undefined)} onCancel={() => setDeleting(null)}>
      <ApiFeedback error={write.error} />
    </ConfirmDialog>
  </>;
}

function Wishes({ wishes, busy, onStart, onDelete }: { wishes: Piece[]; busy: boolean; onStart: (piece: Piece) => void; onDelete: (piece: Piece) => void }) {
  return <div className="wish-inbox"><h3>Życzenia ucznia</h3>
    <ul className="wish-list">{wishes.map((piece) => <li key={piece.id}>
      <span><strong>{piece.title}</strong>{piece.artist ? ` · ${piece.artist}` : ""}</span>
      <span className="row-actions"><ActionButton variant="primary" disabled={busy} onClick={() => onStart(piece)}>Zacznij naukę</ActionButton><button className="text-button danger-button" onClick={() => onDelete(piece)}>Usuń</button></span>
    </li>)}</ul>
  </div>;
}

function PieceRow({ piece, disabled, onStatus, onDelete }: { piece: Piece; disabled: boolean; onStatus: (status: PieceStatus) => void; onDelete: () => void }) {
  return <article className="piece-row">
    <div><strong>{piece.title}</strong>{piece.artist ? <span className="supporting-copy"> · {piece.artist}</span> : null}<p className="lesson-meta">{piece.material_count ? arrangementCount(piece.material_count) : "Brak opracowań"}</p></div>
    <div className="row-actions">
      <select aria-label={`Status utworu ${piece.title}`} value={piece.status} disabled={disabled} onChange={(event) => onStatus(event.target.value as PieceStatus)}>{statusOptions.map((status) => <option key={status} value={status}>{pieceStatusCopy.teacher[status]}</option>)}</select>
      <button className="text-button danger-button" onClick={onDelete}>Usuń</button>
    </div>
  </article>;
}
