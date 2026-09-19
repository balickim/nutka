// Lists the learner wishes and adds a new one. The learner can remove only an own wish that the teacher has not started yet.

import { useState } from "react";

import { pieceStatusCopy } from "../../../api/copy";
import { createPiece, deletePiece, pieceLimits, type Piece } from "../../../api/pieces";
import { ActionButton } from "../../../components/action-button";
import { ApiFeedback } from "../../../components/api-feedback";
import { useToast } from "../../../components/toast";
import { usePieceMutation } from "../../../query/pieces";

export function WishSection({ assignmentId, wishes }: { assignmentId: string; wishes: Piece[] }) {
  const remove = usePieceMutation();
  const { notify } = useToast();
  const [removing, setRemoving] = useState<string | null>(null);
  async function removeWish(piece: Piece) {
    setRemoving(piece.id);
    await remove.mutateAsync({ assignmentId, write: () => deletePiece("learner", piece.id) });
    notify(`Usunięto „${piece.title}” z listy życzeń.`);
  }
  return <section className="panel-section wishes">
    <h2>{pieceStatusCopy.learner.wish}</h2>
    <p className="supporting-copy">Dopisz utwór, który chcesz zagrać. Nauczyciel zobaczy go w swoim panelu i przygotuje opracowanie na Twoim poziomie.</p>
    {wishes.length ? <ul className="wish-list">{wishes.map((piece) => <li key={piece.id}>
      <span><strong>{piece.title}</strong>{piece.artist ? ` · ${piece.artist}` : ""}</span>
      {piece.proposed_by === "learner" ? <ActionButton variant="text" busy={remove.isPending && removing === piece.id} onClick={() => void removeWish(piece).catch(() => undefined)}>Usuń</ActionButton> : null}
    </li>)}</ul> : null}
    <ApiFeedback error={remove.error} />
    <WishForm assignmentId={assignmentId} />
  </section>;
}

function WishForm({ assignmentId }: { assignmentId: string }) {
  const save = usePieceMutation();
  const { notify } = useToast();
  const [title, setTitle] = useState("");
  const [artist, setArtist] = useState("");
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    await save.mutateAsync({ assignmentId, write: () => createPiece("learner", assignmentId, { title, artist }) });
    notify(`Dodano „${title.trim()}” do listy życzeń.`);
    setTitle(""); setArtist("");
  }
  return <form className="inline-form wish-form" onSubmit={(event) => void submit(event).catch(() => undefined)}>
    <label htmlFor="wish-title">Tytuł utworu</label>
    <input id="wish-title" value={title} maxLength={pieceLimits.textMax} required onChange={(event) => setTitle(event.target.value)} />
    <label htmlFor="wish-artist">Wykonawca lub kompozytor (opcjonalnie)</label>
    <input id="wish-artist" value={artist} maxLength={pieceLimits.textMax} onChange={(event) => setArtist(event.target.value)} />
    <ApiFeedback error={save.error} />
    <ActionButton variant="primary" type="submit" busy={save.isPending} disabled={!title.trim()}>Dodaj do listy</ActionButton>
  </form>;
}
