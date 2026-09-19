// Shows the repertoire of the selected assignment: pieces by status with their arrangement versions, learner wishes, and other materials.

import { useQuery } from "@tanstack/react-query";

import { learnerPiecesHelp, pieceStatusCopy } from "../../api/copy";
import { ApiFeedback } from "../../components/api-feedback";
import { EmptyState } from "../../components/empty-state";
import { HelpHeading } from "../../components/help-heading";
import { MaterialList } from "../../components/material-list";
import { Skeleton } from "../../components/skeleton";
import { materialsQuery } from "../../query/materials";
import { piecesQuery } from "../../query/pieces";
import { LearnerShell } from "./learner-shell";
import { PieceCard } from "./pieces/piece-card";
import { groupRepertoire } from "./pieces/repertoire";
import { WishSection } from "./pieces/wish-section";

export function PiecesView() {
  return <LearnerShell lede="Utwory, których się uczysz, i materiały od nauczyciela.">{(context) => <Repertoire accountId={context.accountId} assignmentId={context.assignment.id} />}</LearnerShell>;
}

function Repertoire({ accountId, assignmentId }: { accountId: string; assignmentId: string }) {
  const pieces = useQuery(piecesQuery("learner", accountId, assignmentId));
  const materials = useQuery(materialsQuery("learner", accountId, assignmentId));
  const failure = pieces.error ?? materials.error;
  if (failure) return <section className="panel-section"><ApiFeedback error={failure} onRetry={() => { void pieces.refetch(); void materials.refetch(); }} /></section>;
  if (!pieces.data || !materials.data) return <section className="panel-section"><Skeleton lines={4} label="Ładowanie utworów…" /></section>;
  const { wishes, groups, versions, loose } = groupRepertoire(pieces.data.items, materials.data.items);
  const filled = groups.filter((group) => group.pieces.length > 0);
  return <>
    <section className="panel-section repertoire">
      <HelpHeading title="Moje utwory" help={learnerPiecesHelp} />
      {filled.length === 0 ? <EmptyState>Nauczyciel nie dodał jeszcze utworów. Napisz poniżej, co chcesz zagrać.</EmptyState> : null}
      {filled.map((group) => <div className="piece-group" key={group.status}>
        <h3>{pieceStatusCopy.learner[group.status]}</h3>
        <div className="piece-list">{group.pieces.map((piece) => <PieceCard key={piece.id} piece={piece} versions={versions.get(piece.id) ?? []} />)}</div>
      </div>)}
    </section>
    <WishSection assignmentId={assignmentId} wishes={wishes} />
    {loose.length ? <section className="panel-section learner-materials"><h2>Inne materiały</h2><MaterialList items={loose} empty="" /></section> : null}
  </>;
}
