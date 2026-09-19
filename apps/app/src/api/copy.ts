// Maps English machine values to Polish presentation copy without changing values sent to the backend.

import type {
  ContractStatus,
  CorrectionReason,
  EventType,
  FinancialEntryType,
  LessonOutcomeState,
  PlanType,
  Policy,
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
  package_token_reserved: "Zarezerwowano lekcję z pakietu",
  package_token_used: "Wykorzystano lekcję z pakietu",
  package_token_returned: "Lekcja wróciła do pakietu",
  package_token_expired: "Lekcja z pakietu wygasła",
  package_token_extended: "Przedłużono ważność pakietu",
  package_token_invalidated: "Unieważniono lekcję z pakietu",
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
  credit_applied: "Rozliczono nadpłatę",
  credit_created: "Zapisano nadpłatę",
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
  credit_created: "Zapisano nadpłatę",
  credit_applied: "Rozliczono nadpłatę",
  refund: "Zapisano zwrot",
  correction: "Zapisano korektę",
};

export const errorCopy: Record<string, string> = {
  unauthenticated: "Zaloguj się ponownie, aby kontynuować.",
  unauthorized: "Nie masz dostępu do tej czynności.",
  missing_intent: "Nie udało się tego potwierdzić. Spróbuj ponownie.",
  invalid_request: "Sprawdź wprowadzone dane.",
  invalid_material: "Materiał wymaga tytułu oraz treści lub załącznika. Załączniki to zdjęcia lub PDF do 10 MB.",
  invalid_lesson_note: "Notatka potrzebuje treści. Możesz dołączyć najwyżej 10 materiałów tego ucznia.",
  lesson_not_started: "Notatkę można dodać dopiero po rozpoczęciu lekcji.",
  lesson_cancelled: "Do odwołanej lekcji nie można dodać notatki.",
  invalid_piece: "Utwór potrzebuje tytułu. Tytuł i wykonawca mogą mieć najwyżej 200 znaków.",
  piece_locked: "Możesz usunąć tylko własne życzenie, zanim nauczyciel zacznie z Tobą ten utwór.",
  assignment_inactive: "Zajęcia z tym nauczycielem są zakończone. Możesz tylko przeglądać zapisane treści.",
  invalid_payment_details: "Sprawdź numer konta. Podaj polski numer konta z 26 cyframi i nazwę odbiorcy.",
  invalid_duration: "Lekcja musi mieć czas trwania określony przez bieżące zasady.",
  invalid_grid: "Wybierz inną godzinę rozpoczęcia.",
  conflict: "Wybrany termin koliduje z inną lekcją.",
  lesson_conflict: "Wybrany termin koliduje z inną lekcją.",
  horizon: "Ten termin jest zbyt odległy. Wybierz bliższy.",
  duration_override: "Czas lekcji określają bieżące zasady i nie można go zmienić.",
  plan_precedence: "Wybrany plan nie jest dostępny dla tej rezerwacji.",
  learner_change_cutoff: "Na zmianę terminu jest już za późno.",
  booking_minimum: "Wybierz termin z odpowiednim wyprzedzeniem.",
  change_cutoff: "Na zmianę terminu jest już za późno.",
  regular_contract_active:
    "Aktywny plan regularny blokuje elastyczną rezerwację.",
  package_token_available: "Najpierw wykorzystaj lekcje z pakietu.",
  package_token_exhausted: "W pakiecie nie ma już wolnych lekcji.",
  token_exhausted: "W pakiecie nie ma już wolnych lekcji.",
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
  stale_preview: "Dostępność zmieniła się w międzyczasie. Sprawdź zmianę ponownie.",
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
  internal_error: "Coś poszło nie tak. Spróbuj ponownie.",
  request_failed: "Coś poszło nie tak. Spróbuj ponownie.",
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
  return errorCopy[value] || "Coś poszło nie tak. Spróbuj ponownie.";
}
export function polishCorrectionReason(value: string): string {
  const labels: Record<CorrectionReason, string> = {
    outcome_correction: "Korekta wyniku",
    settlement_correction: "Korekta rozliczenia",
    entitlement_correction: "Korekta dostępnych lekcji",
    ownership_correction: "Korekta przypisania",
    backdated_contract: "Wsteczna aktywacja planu",
    early_contract_end: "Wcześniejsze zakończenie planu",
    token_correction: "Korekta lekcji z pakietu",
    other: "Inna przyczyna",
  };
  return labels[value as CorrectionReason] || unknown;
}

// Explains the availability screen sections in everyday words. Each text restates rules from docs/constitutions/scheduling.md.
export function availabilityHelpCopy(policy: Policy) {
  const horizon = policy.booking_horizon_days;
  return {
    timezone: "Strefa czasowa Twojego kalendarza. Wszystkie godziny dostępności i lekcji podajemy według czasu w tej strefie.",
    weeklyPlan: `Bez reguł cały tydzień jest niedostępny. Każda włączona reguła otwiera godziny w jednym dniu tygodnia i obowiązuje co tydzień, bez daty końca. Przed zapisem zobaczysz skutki zmiany. Każdą kolidującą lekcję w najbliższych ${horizon} dniach musisz odwołać albo przełożyć.`,
    exceptions: "Wyjątek dotyczy konkretnego okresu. „Dostępny” dodaje godziny poza planem tygodniowym. „Niedostępny” zabiera godziny i ma pierwszeństwo przed planem. Nutka nie uwzględnia świąt ani dni wolnych sama, więc dodaj je jako wyjątki.",
    nearTerm: `Lekcje, które zaczynają się w najbliższych ${horizon} dniach. Tak daleko naprzód można rezerwować. Uczeń rezerwuje co najmniej ${policy.learner_booking_minimum_hours} godz. przed lekcją, a Ty możesz później. Każda lekcja blokuje też ${policy.participant_buffer_minutes} minut przed i po sobie. Odwołana lekcja nie wróci po przywróceniu dostępności.`,
    laterContract: `Lekcje ze stałych umów, które zaczynają się później niż za ${horizon} dni. Już rezerwują Twój czas. Gdy zmiana dostępności je obejmie, zostaną pominięte bez opłaty i bez zużycia limitów ucznia. Po przywróceniu dostępności wrócą do kalendarza.`,
  };
}

// Explains the learner panel sections in everyday words. Texts restate docs/constitutions/scheduling.md and docs/api/materials.md.
export const pieceStatusCopy = {
  learner: { wish: "Chcę zagrać", learning: "Uczę się", playing: "Gram", repertoire: "W repertuarze" },
  teacher: { wish: "Życzenie ucznia", learning: "W nauce", playing: "Gra", repertoire: "W repertuarze" },
} as const;

export const learnerPiecesHelp = "Utwory prowadzi Twój nauczyciel. Przy każdym utworze są jego opracowania, od najnowszego. Utwór, który chcesz zagrać, dopisz w sekcji „Chcę zagrać”.";

export const learnerMaterialsHelp = "Materiały dodaje Twój nauczyciel: tekst, zdjęcia i pliki PDF. Najnowsze są na górze. Widzisz tylko materiały od wybranego nauczyciela.";

export function learnerHelpCopy(policy: Policy) {
  const cutoff = policy.learner_booking_minimum_hours;
  return {
    upcoming: `Lekcje, które zaczynają się w najbliższych ${policy.booking_horizon_days} dniach. Możesz przełożyć lub odwołać lekcję, która jeszcze się nie zaczęła. Przełożenie jest możliwe najpóźniej ${cutoff} godz. przed lekcją. Odwołanie później niż ${cutoff} godz. przed lekcją może być płatne albo wykorzystać lekcję z pakietu.`,
    plan: `Każda lekcja należy do jednego planu: stałej umowy, pakietu albo pojedynczej lekcji. Przy stałej umowie terminy wynikają z umowy. Umowa daje ${policy.contract_monthly_reschedules} przełożenie w miesiącu i ${policy.contract_free_cancellations} bezpłatne odwołania. Bez umowy rezerwacja najpierw wykorzystuje wolną lekcję z pakietu, a potem jest pojedynczą lekcją.`,
  };
}

// Polish nouns take one of three forms after a number: 1 lekcja, 2–4 lekcje, 5 lekcji, but 22 lekcje and 12 lekcji.
function polishCount(count: number, [one, few, many]: [string, string, string]): string {
  const tens = count % 100;
  const ones = count % 10;
  if (count === 1) return `1 ${one}`;
  if (ones >= 2 && ones <= 4 && (tens < 12 || tens > 14)) return `${count} ${few}`;
  return `${count} ${many}`;
}

export const lessonCount = (count: number) => polishCount(count, ["lekcja", "lekcje", "lekcji"]);
export const arrangementCount = (count: number) => polishCount(count, ["opracowanie", "opracowania", "opracowań"]);
