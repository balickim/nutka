// Defines immutable history and financial-entry machine values exposed by role-scoped APIs.

export type FinancialEntryType =
  | "charge_created"
  | "package_purchase"
  | "settlement_paid"
  | "settlement_unpaid"
  | "settlement_not_applicable"
  | "adjustment"
  | "credit_created"
  | "credit_applied"
  | "refund"
  | "correction";

export type EventType =
  | "lesson_created"
  | "lesson_converted"
  | "lesson_rescheduled"
  | "lesson_cancelled"
  | "lesson_outcome_recorded"
  | "settlement_changed"
  | "package_purchased"
  | "package_token_reserved"
  | "package_token_used"
  | "package_token_returned"
  | "package_token_expired"
  | "package_token_extended"
  | "package_token_invalidated"
  | "package_validity_extended"
  | "package_closed"
  | "contract_activated"
  | "contract_schedule_changed"
  | "contract_occurrence_rescheduled"
  | "contract_occurrence_omitted"
  | "contract_occurrence_restored"
  | "contract_amended"
  | "contract_notice_submitted"
  | "contract_renewed"
  | "contract_ended"
  | "charge_created"
  | "charge_adjusted"
  | "payment_recorded"
  | "credit_created"
  | "credit_applied"
  | "refund_recorded"
  | "availability_consequence"
  | "availability_changed"
  | "administrative_correction";

export type HistoryEvent = {
  id: string;
  assignment: string;
  aggregate_type: string;
  aggregate_id: string;
  event_type: EventType;
  actor_role: "teacher" | "learner" | "system";
  actor_id?: string;
  event_at: string;
  related_ids?: Record<string, string>;
  prior_state?: Record<string, unknown>;
  new_state?: Record<string, unknown>;
  reason?: string;
  corrects_event?: string;
};
