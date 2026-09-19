// Adds a piece for one learner with a title, an optional artist, and a starting status.

import { useState } from "react";

import { pieceStatusCopy } from "../../../../api/copy";
import { createPiece, pieceLimits, type PieceStatus } from "../../../../api/pieces";
import { ActionButton } from "../../../../components/action-button";
import { ApiFeedback } from "../../../../components/api-feedback";
import { useToast } from "../../../../components/toast";
import { usePieceMutation } from "../../../../query/pieces";

const startStatuses: PieceStatus[] = ["learning", "playing", "repertoire"];

export function PieceForm({ assignmentId }: { assignmentId: string }) {
  const save = usePieceMutation();
  const { notify } = useToast();
  const [title, setTitle] = useState("");
  const [artist, setArtist] = useState("");
  const [status, setStatus] = useState<PieceStatus>("learning");
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    await save.mutateAsync({ assignmentId, write: () => createPiece("teacher", assignmentId, { title, artist, status }) });
    notify(`Dodano utwór „${title.trim()}”.`);
    setTitle(""); setArtist(""); setStatus("learning");
  }
  return <form className="inline-form piece-form" onSubmit={(event) => void submit(event).catch(() => undefined)}>
    <h3>Nowy utwór</h3>
    <label htmlFor="piece-title">Tytuł</label>
    <input id="piece-title" value={title} maxLength={pieceLimits.textMax} required onChange={(event) => setTitle(event.target.value)} />
    <label htmlFor="piece-artist">Wykonawca lub kompozytor</label>
    <input id="piece-artist" value={artist} maxLength={pieceLimits.textMax} onChange={(event) => setArtist(event.target.value)} />
    <label htmlFor="piece-status">Status</label>
    <select id="piece-status" value={status} onChange={(event) => setStatus(event.target.value as PieceStatus)}>{startStatuses.map((value) => <option key={value} value={value}>{pieceStatusCopy.teacher[value]}</option>)}</select>
    <ApiFeedback error={save.error} />
    <ActionButton variant="primary" type="submit" busy={save.isPending} disabled={!title.trim()}>Dodaj utwór</ActionButton>
  </form>;
}
