// Defines role-scoped commercial, ledger, history, calendar, and availability response contracts.

import type {
  ActorRole,
  Assignment,
  ContractOccurrence,
  ContractStatus,
  Currency,
  EventType,
  FinancialEntryType,
  HistoryEvent,
  Lesson,
  LessonOutcome,
  Package,
  PersonaRole,
  PlanType,
  RegularContract,
  ScheduleState,
  SettlementState,
} from "./contracts";

export type PaymentSummary = {
  pending: number;
  intentionally_unpaid: number;
  overdue: number;
  credit_minor: number;
  currency: Currency;
};
export type CommercialSummary = {
  assignment: string;
  active_plan: PlanType | null;
  package?: {
    id: string;
    valid_through: string;
    token_balance: {
      available: number;
      reserved: number;
      used: number;
      expired: number;
      invalidated: number;
    };
  } | null;
  contract?: {
    id: string;
    status: ContractStatus;
    start_on: string;
    end_on: string;
    remaining_monthly_reschedules: number;
    remaining_free_cancellations: number;
    price_minor: number;
    currency: Currency;
  } | null;
  payments: PaymentSummary;
};
export type Charge = {
  id: string;
  assignment: string;
  source_type: "regular_contract" | "ad_hoc";
  source_id: string;
  period: string;
  original_amount_minor: number;
  current_amount_minor: number;
  currency: Currency;
  settlement_state: "pending" | "paid" | "intentionally_unpaid";
  derived_state: string;
  overdue: boolean;
  due_on: string;
  paid_at?: string;
};
export type FinancialEntry = {
  id: string;
  assignment: string;
  charge?: string;
  entry_type: FinancialEntryType;
  amount_minor: number;
  currency: Currency;
  effective_on: string;
  related_lesson?: string;
  related_entry?: string;
  actor_role: ActorRole;
  event_at: string;
};
export type ContractMonth = {
  id: string;
  contract: string;
  assignment: string;
  month: string;
  billable_count?: number;
  forecast_amount_minor: number;
  current_amount_minor?: number;
  currency?: Currency;
  forecast?: boolean;
  charge?: string | null;
  settlement_state?: SettlementState;
  derived_state?: string;
  due_on?: string;
  generated_at?: string;
  occurrence_ids?: string[];
};
export type BusinessEvent = {
  id: string;
  assignment: string;
  aggregate_type: string;
  aggregate_id: string;
  event_type: EventType;
  actor_role: ActorRole;
  actor_id?: string;
  event_at: string;
  related_ids?: Record<string, string>;
  prior_state?: Record<string, unknown>;
  new_state?: Record<string, unknown>;
  reason?: string;
  corrects_event?: string;
};
export type TeacherHistoryEvent = BusinessEvent & { internal_note?: string };
export type AvailabilityRule = {
  id: string;
  teacher: string;
  weekday: number;
  start_time: string;
  end_time: string;
  enabled: boolean;
};
export type ExceptionKind = "available" | "unavailable";
export type AvailabilityException = {
  id: string;
  teacher: string;
  start_at: string;
  end_at: string;
  kind: ExceptionKind;
  note?: string;
  enabled: boolean;
};
export type CalendarResponse = {
  assignments: Assignment[];
  availability_rules: AvailabilityRule[];
  availability_exceptions: AvailabilityException[];
  commercial_summaries: CommercialSummary[];
  near_term_lessons: Lesson[];
  later_contract_lessons?: LaterContractLesson[];
  payment_summary?: PaymentSummary[];
  history_summary?: Array<{ assignment: string; event_count: number }>;
  unresolved_work?: {
    awaiting_outcome: number;
    pending_settlement: number;
    unpaid_charges: number;
    overdue_charges: number;
  };
};
export type LaterContractLesson = Lesson;
export type SlotResponse = {
  teacher: string;
  assignment: string;
  policy_version: string;
  slots: Array<{
    start_at: string;
    end_at: string;
    protected_interval: { start_at: string; end_at: string };
    duration_minutes: number;
  }>;
};
export type AvailabilityProposal = {
  operation: "create" | "update" | "enable" | "disable" | "delete";
  target: "recurring_rule" | "exception";
  rule?: Partial<Omit<AvailabilityRule, "id" | "teacher">>;
  exception?: Partial<Omit<AvailabilityException, "id" | "teacher">>;
  id?: string;
};
export type AvailabilityConflictResolution = {
  lesson: string;
  action: "cancel" | "reschedule";
  replacement_start_at?: string;
};
export type AvailabilityPreview = {
  proposal: AvailabilityProposal;
  near_term_conflicts: Array<{
    lesson: string;
    plan: PlanType;
    start_at: string;
    allowed_resolutions: Array<"cancel" | "reschedule">;
  }>;
  distant_effects: Array<{
    occurrence: string;
    effect: "omit" | "restore";
    start_at: string;
  }>;
  preview_version: string;
};
export type AvailabilityCommitResponse = {
  preview: AvailabilityPreview;
  availability_rule?: AvailabilityRule;
  availability_exception?: AvailabilityException;
};
export type AvailabilityCommitRequest = {
  proposal: AvailabilityProposal;
  preview_version: string;
  resolutions: AvailabilityConflictResolution[];
};
export type CommercialSummaryResponse = CommercialSummary;
export type ContractSeriesResponse = {
  contract?: RegularContract;
  near_term: ContractOccurrence[];
  later: ContractOccurrence[];
  next?: string;
};
export type FinancialWorkResponse = {
  charges: Charge[];
  entries: FinancialEntry[];
  credits: FinancialEntry[];
  next?: string;
};
export type FinancialSummary = {
  charges: Charge[];
  credits: FinancialEntry[];
  pending_settlement: number;
  unpaid_ad_hoc: number;
  unpaid_charges: number;
  overdue_charges: number;
  open_credit_minor: number;
  currency: Currency;
};
export type HistoryResponse<TEvent extends HistoryEvent = HistoryEvent> = {
  items: TEvent[];
  page: number;
  per_page: number;
  total: number;
};
export type FinancialHistoryResponse = {
  items: FinancialEntry[];
  page: number;
  per_page: number;
  total: number;
};
export type UnresolvedWorkResponse = {
  awaiting_outcome: Array<{
    lesson: string;
    assignment: string;
    ended_at: string;
  }>;
  pending_settlements: Array<{
    lesson: string;
    assignment: string;
    amount_minor: number;
    currency: Currency;
  }>;
  unpaid_charges: Charge[];
  overdue_charges: Charge[];
  next?: string;
};
