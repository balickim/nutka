// Groups the pieces of one assignment by status and attaches the arrangement versions of each piece.

import type { Material } from "../../../api/materials";
import type { Piece, PieceStatus } from "../../../api/pieces";

export type Version = { material: Material; number: number };
export type StatusGroup = { status: Exclude<PieceStatus, "wish">; pieces: Piece[] };
export type Repertoire = { wishes: Piece[]; groups: StatusGroup[]; versions: Map<string, Version[]>; loose: Material[] };

const groupOrder: StatusGroup["status"][] = ["learning", "playing", "repertoire"];

// Version numbers follow creation order. Each list shows the newest version first.
// A material of a piece that is not in the list shows with the other materials.
export function groupRepertoire(pieces: Piece[], materials: Material[]): Repertoire {
  const known = new Set(pieces.map((piece) => piece.id));
  const versions = new Map<string, Version[]>();
  const loose: Material[] = [];
  const oldestFirst = [...materials].sort((left, right) => left.created_at.localeCompare(right.created_at));
  for (const material of oldestFirst) {
    if (!material.piece || !known.has(material.piece)) { loose.push(material); continue; }
    const list = versions.get(material.piece) ?? [];
    list.push({ material, number: list.length + 1 });
    versions.set(material.piece, list);
  }
  versions.forEach((list) => list.reverse());
  return {
    wishes: pieces.filter((piece) => piece.status === "wish"),
    groups: groupOrder.map((status) => ({ status, pieces: pieces.filter((piece) => piece.status === status) })),
    versions,
    loose: loose.reverse(),
  };
}
