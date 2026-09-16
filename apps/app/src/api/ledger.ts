// Provides role-scoped financial, payment, unresolved-work, and business-history endpoints through the shared transport.
import { apiRequest } from "./transport";
import type {
  Charge,
  ContractMonth,
  FinancialSummary,
  FinancialHistoryResponse,
  FinancialWorkResponse,
  HistoryEvent,
  HistoryResponse,
  UnresolvedWorkResponse,
  PersonaRole,
} from "./contracts";
const part = (value: string) => encodeURIComponent(value);
const realm = (role: PersonaRole) =>
  role === "teacher" ? "teachers" : "learners";

export function fetchFinancialWork(
  role: "teacher",
  assignmentId?: string,
  signal?: AbortSignal,
): Promise<FinancialWorkResponse>;
export function fetchFinancialWork(
  role: "learner",
  assignmentId?: never,
  signal?: AbortSignal,
): Promise<FinancialSummary>;
export function fetchFinancialWork(
  role: PersonaRole,
  assignmentId?: string,
  signal?: AbortSignal,
): Promise<FinancialWorkResponse | FinancialSummary> {
  if (role === "learner")
    return apiRequest<FinancialSummary>("/api/learners/financial-summary", {
      signal,
    });
  const query = assignmentId ? `?assignment=${part(assignmentId)}` : "";
  return apiRequest<FinancialWorkResponse>(
    `/api/teachers/financial-work${query}`,
    { signal },
  );
}
export const fetchFinancialSummary = (signal?: AbortSignal) =>
  apiRequest<FinancialSummary>("/api/learners/financial-summary", { signal });
export const fetchFinancialHistory = (
  role: PersonaRole,
  assignmentId: string,
  signal?: AbortSignal,
) =>
  apiRequest<FinancialHistoryResponse>(
    `/api/${realm(role)}/assignments/${part(assignmentId)}/financial-history`,
    { signal },
  );
export const fetchContractMonths = (
  role: PersonaRole,
  contractId: string,
  signal?: AbortSignal,
) =>
  apiRequest<{ months: ContractMonth[] }>(
    `/api/${realm(role)}/contracts/${part(contractId)}/months`,
    { signal },
  );
export const recordChargePayment = (
  chargeId: string,
  body: { settlement: "paid" | "intentionally_unpaid" },
) =>
  apiRequest<Charge>(`/api/teachers/charges/${part(chargeId)}/payment`, {
    method: "POST",
    body,
  });
export const recordChargeRefund = (
  chargeId: string,
  body: { amount_minor: number; reason: string },
) =>
  apiRequest<Charge>(`/api/teachers/charges/${part(chargeId)}/refund`, {
    method: "POST",
    body,
  });
export const correctCharge = (
  chargeId: string,
  body: { reason: string; note?: string },
) =>
  apiRequest<Charge>(`/api/teachers/charges/${part(chargeId)}/correction`, {
    method: "POST",
    body,
  });
export const fetchUnresolvedWork = (signal?: AbortSignal) =>
  apiRequest<UnresolvedWorkResponse>("/api/teachers/unresolved-work", {
    signal,
  });
export function fetchHistory(
  role: "teacher",
  assignmentId: string,
  signal?: AbortSignal,
): Promise<HistoryResponse<HistoryEvent & { internal_note?: string }>>;
export function fetchHistory(
  role: "learner",
  assignmentId: string,
  signal?: AbortSignal,
): Promise<HistoryResponse<HistoryEvent>>;
export async function fetchHistory(
  role: PersonaRole,
  assignmentId: string,
  signal?: AbortSignal,
): Promise<HistoryResponse<HistoryEvent>> {
  const result = await apiRequest<
    HistoryResponse<HistoryEvent & { internal_note?: string }>
  >(`/api/${realm(role)}/assignments/${part(assignmentId)}/history`, {
    signal,
  });
  return result;
}
