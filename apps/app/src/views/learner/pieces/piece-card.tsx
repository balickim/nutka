// Shows one piece with its arrangement versions, newest first, behind one disclosure.

import { arrangementCount } from "../../../api/copy";
import type { Piece } from "../../../api/pieces";
import { MaterialList } from "../../../components/material-list";
import type { Version } from "./repertoire";

export function PieceCard({ piece, versions }: { piece: Piece; versions: Version[] }) {
  const numbers = new Map(versions.map((version) => [version.material.id, version.number]));
  return <article className="piece-card">
    <div className="piece-heading"><h4>{piece.title}</h4>{piece.artist ? <p className="supporting-copy">{piece.artist}</p> : null}</div>
    {versions.length === 0
      ? <p className="lesson-meta">Nauczyciel nie dodał jeszcze opracowania.</p>
      : <details className="piece-versions"><summary>Pokaż {arrangementCount(versions.length)}</summary>
        <MaterialList items={versions.map((version) => version.material)} empty="" caption={(material) => `Wersja ${numbers.get(material.id) ?? ""}`} />
      </details>}
  </article>;
}
