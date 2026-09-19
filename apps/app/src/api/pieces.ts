// Declares the piece contract and requests the teacher piece writes, the learner wish writes, and the persona piece lists.

import type { PersonaRole } from "./contracts";
import { apiRequest } from "./transport";

export type PieceStatus = "wish" | "learning" | "playing" | "repertoire";

export type Piece = {
  id: string;
  assignment: string;
  title: string;
  artist: string;
  status: PieceStatus;
  status_changed_at: string;
  proposed_by: PersonaRole;
  material_count: number;
  latest_material_at: string | null;
  created_at: string;
};

export type PieceDraft = { title: string; artist: string };

export const pieceLimits = { textMax: 200 };

const part = encodeURIComponent;
const segment = (role: PersonaRole) => (role === "teacher" ? "teachers" : "learners");

export const fetchPieces = (role: PersonaRole, assignmentId: string, signal?: AbortSignal) =>
  apiRequest<{ items: Piece[] }>(`/api/${segment(role)}/assignments/${part(assignmentId)}/pieces`, { signal });

export const createPiece = (role: PersonaRole, assignmentId: string, body: PieceDraft & { status?: PieceStatus }) =>
  apiRequest<Piece>(`/api/${segment(role)}/assignments/${part(assignmentId)}/pieces`, { method: "POST", body });

export const updatePiece = (pieceId: string, body: Partial<PieceDraft & { status: PieceStatus }>) =>
  apiRequest<Piece>(`/api/teachers/pieces/${part(pieceId)}`, { method: "PATCH", body });

export const deletePiece = (role: PersonaRole, pieceId: string) =>
  apiRequest<void>(`/api/${segment(role)}/pieces/${part(pieceId)}`, { method: "DELETE" });
