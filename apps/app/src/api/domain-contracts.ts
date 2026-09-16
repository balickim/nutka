// Defines English plan, entitlement, lesson, assignment, and scheduling DTOs.

import type { TeacherLocalDate, Currency } from "./policy-contracts";

export type PersonaRole = "teacher" | "learner";
export type ActorRole = PersonaRole | "system";
export type PlanType = "regular_contract" | "package" | "ad_hoc";
export type TokenState =
  "available" | "reserved" | "used" | "expired" | "invalidated";
export type ContractStatus = "active" | "ended" | "notice_given";
export type ScheduleState = "scheduled" | "cancelled" | "omitted";
export type LessonOutcome = "completed" | "learner_no_show";
export type LessonOutcomeState = LessonOutcome | "awaiting_outcome";
export type SettlementState =
  "pending_settlement" | "paid" | "intentionally_unpaid" | "not_applicable";
export type ChargeSettlementState = "pending" | "paid" | "intentionally_unpaid";
export type CorrectionReason =
  | "outcome_correction"
  | "settlement_correction"
  | "entitlement_correction"
  | "ownership_correction"
  | "backdated_contract"
  | "early_contract_end"
  | "token_correction"
  | "other";

export type Assignment = {
  id: string;
  teacher: string;
  learner: string;
  teacher_name?: string;
  learner_name?: string;
  active: boolean;
};

export type Token = {
  id: string;
  ordinal: number;
  state: TokenState;
  lesson: string | null;
};

export type Package = {
  id: string;
  assignment: string;
  status: "open" | "closed";
  purchased_on: TeacherLocalDate;
  valid_through: TeacherLocalDate;
  closed_at?: string;
  price_minor: number;
  currency: Currency;
  policy_version: string;
  tokens: Token[];
};

export type ContractOccurrence = {
  id: string;
  contract: string;
  assignment: string;
  original_local_date: TeacherLocalDate;
  original_start_at: string;
  start_at: string;
  end_at: string;
  schedule_state: ScheduleState;
  outcome?: string;
  billing_outcome: string;
  individually_rescheduled: boolean;
  unit_price_minor: number;
  currency: Currency;
};

export type RegularContract = {
  id: string;
  assignment: string;
  status: ContractStatus;
  start_on: TeacherLocalDate;
  end_on: TeacherLocalDate;
  weekday: number;
  start_time: string;
  price_minor: number;
  currency: Currency;
  notice_at?: string;
  effective_end_on?: TeacherLocalDate;
  policy_version: string;
  remaining_monthly_reschedules: number;
  remaining_free_cancellations: number;
};

export type ProtectedInterval = { start_at: string; end_at: string };
export type Lesson = {
  id: string;
  teacher: string;
  learner: string;
  assignment: string;
  start_at: string;
  end_at: string;
  duration_minutes: number;
  plan_type: PlanType;
  package_token: string | null;
  contract: string | null;
  original_local_date?: TeacherLocalDate;
  original_start_at?: string;
  policy_version: string;
  unit_price_minor: number;
  currency: Currency;
  schedule_state: ScheduleState;
  outcome: LessonOutcomeState | null;
  protected_interval: ProtectedInterval;
  cancellation_initiator_role?: PersonaRole;
  cancellation_initiator_id?: string;
  cancelled_at?: string;
};

export type LessonPayment = {
  lesson: string;
  assignment: string;
  amount_minor: number;
  currency: Currency;
  settlement_state: SettlementState;
};

export type Slot = {
  start_at: string;
  end_at: string;
  protected_interval: ProtectedInterval;
  duration_minutes: number;
};
