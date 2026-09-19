// Owns the lesson note list query and the teacher note mutation, delegating cache effects to the central registry.

import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";

import type { PersonaRole } from "../api/contracts";
import { fetchLessonNotes } from "../api/lesson-notes";
import { applyCacheEffect } from "./effects";
import { queryKeys, queryRules } from "./keys";

export function lessonNotesQuery(role: PersonaRole, accountId: string | undefined, assignmentId: string) {
  return queryOptions({
    queryKey: queryKeys.lessonNotes(role, accountId ?? "", assignmentId),
    queryFn: ({ signal }) => fetchLessonNotes(role, assignmentId, signal),
    enabled: Boolean(accountId),
  });
}

export function useNoteMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ write }: { assignmentId: string; write: () => Promise<unknown> }) => write(),
    onSuccess: (_result, { assignmentId }) => applyCacheEffect(client, queryRules.noteWrite(assignmentId)),
  });
}
