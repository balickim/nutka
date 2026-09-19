// Maps English machine values to Polish presentation copy without changing values sent to the backend.

import type {
  ContractStatus,
  CorrectionReason,
  EventType,
  FinancialEntryType,
  LessonOutcomeState,
  PlanType,
  ScheduleState,
  SettlementState,
  TokenState,
} from "./contracts";

const unknown = "Nieznane";

export const planCopy: Record<PlanType, string> = {
  regular_contract: "Plan regularny",
  package: "Pakiet lekcji",
  ad_hoc: "Pojedyncza lekcja",
};
export const tokenStateCopy: Record<TokenState, string> = {
  available: "Dostępny",
  reserved: "Zarezerwowany",
  used: "Wykorzystany",
  expired: "Wygasły",
  invalidated: "Unieważniony",
};
export const contractStatusCopy: Record<ContractStatus, string> = {
  active: "Aktywny",
  ended: "Zakończony",
  notice_given: "W okresie wypowiedzenia",
};
export const scheduleStateCopy: Record<ScheduleState, string> = {
  scheduled: "Zaplanowana",
  cancelled: "Odwołana",
  omitted: "Pominięta",
};
export const billingOutcomeCopy: Record<string, string> = {
  billable: "Płatna",
  planned_omission: "Pominięta bez opłaty",
  free_cancellation: "Bezpłatnie odwołana",
  teacher_cancellation: "Odwołana przez nauczyciela bez opłaty",
  late_cancellation: "Płatne późne odwołanie",
  exhausted_cancellation: "Płatne odwołanie po wykorzystaniu limitu",
  no_show: "Płatna nieobecność ucznia",
  contract_termination: "Usunięta po zakończeniu umowy",
};
export const outcomeCopy: Record<LessonOutcomeState, string> = {
  awaiting_outcome: "Oczekuje na wynik",
  completed: "Zrealizowana",
  learner_no_show: "Nieobecność ucznia",
};
export const settlementCopy: Record<SettlementState, string> = {
  pending_settlement: "Oczekuje na rozliczenie",
  paid: "Opłacona",
  intentionally_unpaid: "Oznaczona jako nieopłacona",
  not_applicable: "Nie dotyczy",
};
export const chargeStateCopy: Record<string, string> = {
  pending: "Oczekuje na płatność",
  paid: "Opłacona",
  intentionally_unpaid: "Oznaczona jako nieopłacona",
  overdue: "Zaległa",
};

export const eventCopy: Record<EventType, string> = {
  lesson_created: "Utworzono lekcję",
  lesson_converted: "Zmieniono plan lekcji",
  lesson_rescheduled: "Przełożono lekcję",
  lesson_cancelled: "Odwołano lekcję",
  lesson_outcome_recorded: "Zapisano wynik lekcji",
  settlement_changed: "Zmieniono rozliczenie",
  package_purchased: "Kupiono pakiet",
  package_closed: "Zamknięto pakiet",
  package_token_reserved: "Zarezerwowano token",
  package_token_used: "Wykorzystano token",
  package_token_returned: "Zwrócono token",
  package_token_expired: "Token wygasł",
  package_token_extended: "Przedłużono ważność tokenu",
  package_token_invalidated: "Unieważniono token",
  package_validity_extended: "Przedłużono ważność pakietu",
  contract_activated: "Aktywowano plan regularny",
  contract_schedule_changed: "Zmieniono harmonogram planu",
  contract_occurrence_rescheduled: "Przełożono lekcję planu regularnego",
  contract_occurrence_omitted: "Pominięto lekcję planu regularnego",
  contract_occurrence_restored: "Przywrócono lekcję planu regularnego",
  contract_amended: "Zmieniono warunki planu",
  contract_notice_submitted: "Złożono wypowiedzenie",
  contract_renewed: "Odnowiono plan regularny",
  contract_ended: "Zakończono plan regularny",
  charge_created: "Utworzono należność",
  charge_adjusted: "Skorygowano należność",
  payment_recorded: "Zapisano płatność",
  credit_applied: "Zastosowano kredyt",
  credit_created: "Utworzono kredyt",
  refund_recorded: "Zapisano zwrot",
  availability_consequence: "Zastosowano skutek dostępności",
  availability_changed: "Zmieniono dostępność",
  administrative_correction: "Zapisano korektę administracyjną",
};

export const financialEntryCopy: Record<FinancialEntryType, string> = {
  charge_created: "Utworzono należność",
  package_purchase: "Kupiono pakiet",
  settlement_paid: "Rozliczono jako opłacone",
  settlement_unpaid: "Oznaczono jako nieopłacone",
  settlement_not_applicable: "Rozliczenie nie dotyczy",
  adjustment: "Skorygowano należność",
  credit_created: "Utworzono kredyt",
  credit_applied: "Zastosowano kredyt",
  refund: "Zapisano zwrot",
  correction: "Zapisano korektę",
};

export const errorCopy: Record<string, string> = {
  unauthenticated: "Zaloguj się ponownie, aby kontynuować.",
  unauthorized: "Nie masz dostępu do tej operacji.",
  missing_intent: "Nie udało się potwierdzić tej operacji. Spróbuj ponownie.",
  invalid_request: "Sprawdź wprowadzone dane.",
  invalid_material: "Materiał wymaga tytułu oraz treści lub załącznika. Załączniki to zdjęcia lub PDF do 10 MB.",
  invalid_duration: "Lekcja musi mieć czas trwania określony przez bieżące zasady.",
  invalid_grid: "Wybierz termin zgodny z bieżącą siatką godzin.",
  conflict: "Wybrany termin koliduje z inną lekcją.",
  lesson_conflict: "Wybrany termin koliduje z inną lekcją.",
  horizon: "Wybierz termin w skonfigurowanym horyzoncie.",
  duration_override: "Czas lekcji określają bieżące zasady i nie można go zmienić.",
  plan_precedence: "Wybrany plan nie jest dostępny dla tej rezerwacji.",
  learner_change_cutoff: "Na zmianę terminu jest już za późno.",
  booking_minimum: "Wybierz termin z odpowiednim wyprzedzeniem.",
  change_cutoff: "Na zmianę terminu jest już za późno.",
  regular_contract_active:
    "Aktywny plan regularny blokuje elastyczną rezerwację.",
  package_token_available: "Dostępny token pakietu musi zostać użyty.",
  package_token_exhausted: "Brak dostępnych tokenów.",
  token_exhausted: "Brak dostępnych tokenów.",
  package_expired: "Pakiet wygasł.",
  package_overlap: "Pakiet koliduje z istniejącym zobowiązaniem.",
  contract_overlap: "Plan regularny koliduje z istniejącym zobowiązaniem.",
  contract_allowance_exhausted: "Wykorzystano limit planu regularnego.",
  allowance_exhausted: "Wykorzystano dostępny limit.",
  unresolved_obligations: "Najpierw rozwiąż istniejące zobowiązania.",
  unresolved_obligation: "Najpierw rozwiąż istniejące zobowiązania.",
  incomplete_resolution: "Rozwiąż wszystkie konflikty przed zapisaniem.",
  invalid_resolution: "Wybrane rozwiązanie konfliktu jest nieprawidłowe.",
  contract_replacement_deadline: "Termin zastępczy przekracza dozwolony okres.",
  lesson_rescheduled: "Ta lekcja została już przełożona.",
  package_has_future_lessons: "Pakiet ma zaplanowane przyszłe lekcje.",
  stale_preview: "Dostępność zmieniła się. Wygeneruj podgląd ponownie.",
  invalid_preview_resolution: "Rozwiąż wszystkie konflikty przed zapisaniem.",
  correction_reason_required: "Korekta wymaga podania przyczyny.",
  correction_not_allowed: "Nie można skorygować tego przejścia.",
  event_immutable: "Historii nie można zmienić ani usunąć.",
  timezone_locked: "Przyszłe zobowiązania blokują zmianę strefy czasowej.",
  invalid_pagination: "Sprawdź parametry stronicowania.",
  invalid_outcome: "Wybierz prawidłowy wynik lekcji.",
  invalid_settlement: "Wybierz prawidłowy stan rozliczenia.",
  future_date: "Data nie może przypadać w przyszłości.",
  started_lesson: "Rozpoczętej lekcji nie można zmienić.",
  not_found: "Nie znaleziono wskazanego zasobu.",
  internal_error: "Nie udało się wykonać operacji.",
  request_failed: "Nie udało się wykonać operacji.",
};

export function polishPlan(value: string): string {
  return planCopy[value as PlanType] || unknown;
}
export function polishTokenState(value: string): string {
  return tokenStateCopy[value as TokenState] || unknown;
}
export function polishEvent(value: string): string {
  return eventCopy[value as EventType] || unknown;
}
export function polishFinancialEntry(value: string): string {
  return financialEntryCopy[value as FinancialEntryType] || unknown;
}
export function polishError(value: string): string {
  return errorCopy[value] || "Nie udało się wykonać operacji.";
}
export function polishCorrectionReason(value: string): string {
  const labels: Record<CorrectionReason, string> = {
    outcome_correction: "Korekta wyniku",
    settlement_correction: "Korekta rozliczenia",
    entitlement_correction: "Korekta uprawnienia",
    ownership_correction: "Korekta własności",
    backdated_contract: "Wsteczna aktywacja planu",
    early_contract_end: "Wcześniejsze zakończenie planu",
    token_correction: "Korekta tokenu",
    other: "Inna przyczyna",
  };
  return labels[value as CorrectionReason] || unknown;
}
