// Owns the piece list query and the piece mutation, delegating cache effects to the central registry.

import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";

import type { PersonaRole } from "../api/contracts";
import { fetchPieces } from "../api/pieces";
import { applyCacheEffect } from "./effects";
import { queryKeys, queryRules } from "./keys";

export function piecesQuery(role: PersonaRole, accountId: string | undefined, assignmentId: string) {
  return queryOptions({
    queryKey: queryKeys.pieces(role, accountId ?? "", assignmentId),
    queryFn: ({ signal }) => fetchPieces(role, assignmentId, signal),
    enabled: Boolean(accountId),
  });
}

export function usePieceMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ write }: { assignmentId: string; write: () => Promise<unknown> }) => write(),
    onSuccess: (_result, { assignmentId }) => applyCacheEffect(client, queryRules.pieceWrite(assignmentId)),
  });
}
