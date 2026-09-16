// Defines policy and money DTOs shared by authenticated commercial API responses.

export type Currency = "PLN";
export type TeacherLocalDate = string;
export type PolicyVersion = string;

export type Money = {
  amount_minor: number;
  currency: Currency;
};

export type Policy = {
  version: PolicyVersion;
  currency: Currency;
  ad_hoc_price_minor: number;
  package_price_minor: number;
  regular_lesson_price_minor: number;
  lesson_duration_minutes: number;
  start_grid_minutes: number;
  participant_buffer_minutes: number;
  learner_booking_minimum_hours: number;
  learner_change_cutoff_hours: number;
  booking_horizon_days: number;
  package_token_count: number;
  package_validity_days: number;
  teacher_cancellation_extension_days: number;
  contract_monthly_reschedules: number;
  contract_free_cancellations: number;
  contract_replacement_deadline_days: number;
  monthly_payment_due_day: number;
  contract_end_month: number;
  contract_end_day: number;
};

export type PolicySnapshot = Pick<
  Policy,
  | "version"
  | "currency"
  | "ad_hoc_price_minor"
  | "package_price_minor"
  | "regular_lesson_price_minor"
  | "lesson_duration_minutes"
  | "start_grid_minutes"
  | "participant_buffer_minutes"
  | "learner_booking_minimum_hours"
  | "learner_change_cutoff_hours"
  | "booking_horizon_days"
  | "package_token_count"
  | "package_validity_days"
  | "teacher_cancellation_extension_days"
  | "contract_monthly_reschedules"
  | "contract_free_cancellations"
  | "contract_replacement_deadline_days"
  | "monthly_payment_due_day"
  | "contract_end_month"
  | "contract_end_day"
>;
