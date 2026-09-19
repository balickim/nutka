// Owns the practice queries and the practice mutation, delegating cache effects to the central registry.

import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";

import type { PersonaRole } from "../api/contracts";
import { fetchDaySummaries, fetchPracticeSessions, fetchPracticeSummary, fetchPracticeTasks, type TaskStatus } from "../api/practice";
import { applyCacheEffect } from "./effects";
import { queryKeys, queryRules } from "./keys";

export function practiceTasksQuery(role: PersonaRole, accountId: string | undefined, assignmentId: string, status: TaskStatus | "all" = "active") {
  return queryOptions({
    queryKey: queryKeys.practiceTasks(role, accountId ?? "", assignmentId, status),
    queryFn: ({ signal }) => fetchPracticeTasks(role, assignmentId, status, signal),
    enabled: Boolean(accountId),
  });
}

export function practiceSessionsQuery(role: PersonaRole, accountId: string | undefined, assignmentId: string, page = 1) {
  return queryOptions({
    queryKey: queryKeys.practiceSessions(role, accountId ?? "", assignmentId, page),
    queryFn: ({ signal }) => fetchPracticeSessions(role, assignmentId, page, signal),
    enabled: Boolean(accountId),
  });
}

export function practiceSummaryQuery(role: PersonaRole, accountId: string | undefined, assignmentId: string) {
  return queryOptions({
    queryKey: queryKeys.practiceSummary(role, accountId ?? "", assignmentId),
    queryFn: ({ signal }) => fetchPracticeSummary(role, assignmentId, signal),
    enabled: Boolean(accountId),
  });
}

export function daySummariesQuery(accountId: string, date: string) {
  return queryOptions({
    queryKey: queryKeys.practiceDay(accountId, date),
    queryFn: ({ signal }) => fetchDaySummaries(date, signal),
  });
}

export function usePracticeMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ write }: { assignmentId: string; write: () => Promise<unknown> }) => write(),
    onSuccess: (_result, { assignmentId }) => applyCacheEffect(client, queryRules.practiceWrite(assignmentId)),
  });
}
