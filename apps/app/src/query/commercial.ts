// Owns commercial query options and mutation hooks while delegating all cache effects to the central registry.

import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";

import {
  fetchBusinessPolicy,
  fetchCommercialSummary,
  fetchContractSeries,
  fetchFinancialWork,
  fetchFinancialSummary,
  fetchHistory,
  fetchUnresolvedWork,
} from "../api/commercial";
import type { PersonaRole } from "../api/contracts";
import { applyCacheEffect } from "./effects";
import { queryKeys, queryRules } from "./keys";

export function businessPolicyQuery(role: PersonaRole) {
  return queryOptions({
    queryKey: queryKeys.policy(role),
    queryFn: ({ signal }) => fetchBusinessPolicy(role, signal),
    staleTime: Infinity,
  });
}

export function commercialSummaryQuery(
  role: PersonaRole,
  accountId: string | undefined,
  assignmentId: string,
) {
  return queryOptions({
    queryKey: queryKeys.commercialSummary(role, accountId ?? "", assignmentId),
    queryFn: ({ signal }) => fetchCommercialSummary(role, assignmentId, signal),
    enabled: Boolean(accountId),
  });
}

export function contractSeriesQuery(
  role: "teacher",
  accountId: string | undefined,
  assignmentId: string,
  contractId: string,
) {
  return queryOptions({
    queryKey: queryKeys.contractSeries(role, assignmentId, contractId),
    queryFn: ({ signal }) => fetchContractSeries(role, contractId, signal),
    enabled: Boolean(accountId),
  });
}

export function financialWorkQuery(
  role: PersonaRole,
  accountId: string | undefined,
  assignmentId?: string,
) {
  if (role === "learner") {
    return queryOptions({
      queryKey: queryKeys.financialWork(role, accountId ?? ""),
      queryFn: ({ signal }) => fetchFinancialSummary(signal),
      enabled: Boolean(accountId),
    });
  }
  return queryOptions({
    queryKey: queryKeys.financialWork(role, accountId ?? "", assignmentId),
    queryFn: ({ signal }) =>
      fetchFinancialWork("teacher", assignmentId, signal),
    enabled: Boolean(accountId),
  });
}

export function historyQuery(
  role: PersonaRole,
  accountId: string | undefined,
  assignmentId: string,
) {
  return queryOptions({
    queryKey: queryKeys.history(role, accountId ?? "", assignmentId),
    queryFn: ({ signal }) =>
      role === "teacher"
        ? fetchHistory("teacher", assignmentId, signal)
        : fetchHistory("learner", assignmentId, signal),
    enabled: Boolean(accountId),
  });
}

export function unresolvedWorkQuery(accountId: string | undefined) {
  return queryOptions({
    queryKey: queryKeys.unresolvedWork("teacher", accountId ?? ""),
    queryFn: ({ signal }) => fetchUnresolvedWork(signal),
    enabled: Boolean(accountId),
  });
}

type Write = () => Promise<unknown>;
type ScopedWrite = { assignmentId?: string; write: Write };

function useScopedWrite(
  effect: (assignmentId?: string) => ReturnType<typeof queryRules.bookingWrite>,
) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ write }: ScopedWrite) => write(),
    onSuccess: (_result, { assignmentId }) =>
      applyCacheEffect(client, effect(assignmentId)),
  });
}

export function useBookingMutation() {
  return useScopedWrite((assignmentId) =>
    queryRules.bookingWrite(assignmentId),
  );
}

export function useLifecycleMutation() {
  return useScopedWrite((assignmentId) =>
    queryRules.lifecycleWrite(assignmentId),
  );
}

export function usePlanMutation() {
  return useScopedWrite((assignmentId) => queryRules.planWrite(assignmentId));
}

export function useAvailabilityCommitMutation() {
  return useScopedWrite((assignmentId) =>
    queryRules.availabilityCommit(assignmentId),
  );
}

export function useOutcomeMutation() {
  return useScopedWrite((assignmentId) =>
    queryRules.outcomeWrite(assignmentId),
  );
}

export function useSettlementMutation() {
  return useScopedWrite((assignmentId) =>
    queryRules.settlementWrite(assignmentId),
  );
}

export function useCorrectionMutation() {
  return useScopedWrite((assignmentId) =>
    queryRules.correctionWrite(assignmentId),
  );
}
