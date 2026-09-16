// Provides policy, assignment, package, and regular-contract endpoints through the shared transport.
import { apiRequest } from "./transport";
import type {
  Assignment,
  CommercialSummary,
  ContractSeriesResponse,
  Package,
  PersonaRole,
  RegularContract,
  Token,
  Policy,
  CalendarResponse,
} from "./contracts";
const part = (value: string) => encodeURIComponent(value);
const realm = (role: PersonaRole) =>
  role === "teacher" ? "teachers" : "learners";

export const fetchBusinessPolicy = (role: PersonaRole, signal?: AbortSignal) =>
  apiRequest<Policy>(`/api/${realm(role)}/business-policy`, { signal });
export const fetchCalendar = (role: PersonaRole, signal?: AbortSignal) =>
  apiRequest<CalendarResponse>(`/api/${realm(role)}/calendar`, { signal });
export const fetchAssignments = (role: PersonaRole, signal?: AbortSignal) =>
  apiRequest<{ assignments: Assignment[] }>(`/api/${realm(role)}/assignments`, {
    signal,
  });
export const updateAssignment = (assignmentId: string, body: { active: boolean }) =>
  apiRequest<Assignment>(
    `/api/teachers/assignments/${part(assignmentId)}`,
    { method: "PATCH", body },
  );
export const fetchCommercialSummary = (
  role: PersonaRole,
  assignmentId: string,
  signal?: AbortSignal,
) =>
  apiRequest<CommercialSummary>(
    `/api/${realm(role)}/assignments/${part(assignmentId)}/commercial-summary`,
    { signal },
  );
export const fetchPackages = (
  role: PersonaRole,
  assignmentId: string,
  signal?: AbortSignal,
) =>
  apiRequest<{ packages: Package[] }>(
    `/api/${realm(role)}/assignments/${part(assignmentId)}/packages`,
    { signal },
  );

export type PurchasePackageRequest = {
  purchased_on?: string;
  convert_lesson_ids?: string[];
};
export const purchasePackage = (
  assignmentId: string,
  body: PurchasePackageRequest,
) =>
  apiRequest<Package>(
    `/api/teachers/assignments/${part(assignmentId)}/packages`,
    { method: "POST", body },
  );
export type ClosePackageRequest = {
  reason: string;
  refund?: {
    amount_minor: number;
    currency: "PLN";
    note?: string;
  };
};
export const closePackage = (packageId: string, body: ClosePackageRequest) =>
  apiRequest<Package>(`/api/teachers/packages/${part(packageId)}/close`, {
    method: "POST",
    body,
  });
export type PackageCorrectionRequest = {
  token_id: string;
  target: Token["state"];
  corrects_event: string;
  reason: string;
};
export const correctPackage = (
  packageId: string,
  body: PackageCorrectionRequest,
) =>
  apiRequest<Package>(`/api/teachers/packages/${part(packageId)}/correction`, {
    method: "POST",
    body,
  });

export type ActivateContractRequest = {
  start_on: string;
  weekday: number;
  start_time: string;
  convert_lesson_ids?: string[];
  backdate_reason?: string;
  past_outcomes?: Record<string, string>;
};
export const fetchContracts = (
  role: PersonaRole,
  assignmentId: string,
  signal?: AbortSignal,
) =>
  apiRequest<{ contracts: RegularContract[] }>(
    `/api/${realm(role)}/assignments/${part(assignmentId)}/contracts`,
    { signal },
  );
export const fetchContractSeries = (
  role: "teacher",
  contractId: string,
  signal?: AbortSignal,
) =>
  apiRequest<ContractSeriesResponse>(
    `/api/${realm(role)}/contracts/${part(contractId)}/series`,
    { signal },
  );
export const activateContract = (
  assignmentId: string,
  body: ActivateContractRequest,
) =>
  apiRequest<RegularContract>(
    `/api/teachers/assignments/${part(assignmentId)}/contracts`,
    { method: "POST", body },
  );
export type ContractScheduleRequest = {
  effective_on: string;
  weekday: number;
  start_time: string;
};
export const changeContractSchedule = (
  contractId: string,
  body: ContractScheduleRequest,
) =>
  apiRequest<RegularContract>(
    `/api/teachers/contracts/${part(contractId)}/schedule`,
    { method: "POST", body },
  );
export const submitContractNotice = (role: PersonaRole, contractId: string) =>
  apiRequest<RegularContract>(
    `/api/${realm(role)}/contracts/${part(contractId)}/notice`,
    { method: "POST", body: {} },
  );
export const renewContract = (
  contractId: string,
  body: { start_on: string },
) =>
  apiRequest<RegularContract>(
    `/api/teachers/contracts/${part(contractId)}/renew`,
    { method: "POST", body },
  );
export const amendContractPrice = (
  contractId: string,
  body: { effective_on: string; price_minor: number; currency: "PLN" },
) =>
  apiRequest<RegularContract>(
    `/api/teachers/contracts/${part(contractId)}/amendments`,
    { method: "POST", body },
  );
export const correctContract = (
  contractId: string,
  body: { event_id: string; reason: string },
) =>
  apiRequest<RegularContract>(
    `/api/teachers/contracts/${part(contractId)}/correction`,
    { method: "POST", body },
  );

export const endContractEarly = (
  contractId: string,
  body: { end_on: string; reason: string },
) =>
  apiRequest<RegularContract>(
    `/api/teachers/contracts/${part(contractId)}/early-end`,
    { method: "POST", body },
  );
