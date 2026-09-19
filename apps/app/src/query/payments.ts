// Owns the learner payment-due query and the teacher transfer details query and mutation, delegating cache effects to the central registry.

import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";

import { fetchPaymentDetails, fetchPaymentDue, savePaymentDetails, type PaymentDetails } from "../api/payments";
import { applyCacheEffect } from "./effects";
import { queryKeys, queryRules } from "./keys";

export function paymentDueQuery(accountId: string | undefined, assignmentId: string) {
  return queryOptions({
    queryKey: queryKeys.paymentDue(accountId ?? "", assignmentId),
    queryFn: ({ signal }) => fetchPaymentDue(assignmentId, signal),
    enabled: Boolean(accountId),
  });
}

export function paymentDetailsQuery(accountId: string) {
  return queryOptions({
    queryKey: queryKeys.paymentDetails(accountId),
    queryFn: ({ signal }) => fetchPaymentDetails(signal),
  });
}

export function usePaymentDetailsMutation(accountId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (details: PaymentDetails) => savePaymentDetails(details),
    onSuccess: () => applyCacheEffect(client, queryRules.paymentDetailsWrite(accountId)),
  });
}
