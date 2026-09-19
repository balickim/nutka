// Owns learner material query options and the teacher material mutation, delegating cache effects to the central registry.

import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";

import type { PersonaRole } from "../api/contracts";
import { fetchMaterials } from "../api/materials";
import { applyCacheEffect } from "./effects";
import { queryKeys, queryRules } from "./keys";

export function materialsQuery(role: PersonaRole, accountId: string | undefined, assignmentId: string) {
  return queryOptions({
    queryKey: queryKeys.materials(role, accountId ?? "", assignmentId),
    queryFn: ({ signal }) => fetchMaterials(role, assignmentId, signal),
    enabled: Boolean(accountId),
  });
}

export function useMaterialMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ write }: { assignmentId: string; write: () => Promise<unknown> }) => write(),
    onSuccess: (_result, { assignmentId }) => applyCacheEffect(client, queryRules.materialWrite(assignmentId)),
  });
}
