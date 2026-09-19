// States the learner's active plan, balances, and payment position, so the teacher reads the case before changing it.

import { useQuery } from "@tanstack/react-query";

import { contractStatusCopy, planCopy } from "../../../api/copy";
import type { Assignment, CommercialSummary } from "../../../api/contracts";
import { ApiFeedback } from "../../../components/ApiFeedback";
import { EmptyState } from "../../../components/EmptyState";
import { Skeleton } from "../../../components/Skeleton";
import { formatMoney } from "../../../money";
import { commercialSummaryQuery } from "../../../query/commercial";

export function OverviewTab({ accountId, assignment }: { accountId: string; assignment: Assignment }) {
  const summary = useQuery(commercialSummaryQuery("teacher", accountId, assignment.id));
  if (summary.error) return <ApiFeedback error={summary.error} onRetry={() => void summary.refetch()} />;
  if (summary.isPending || !summary.data) return <Skeleton lines={5} label="Ładowanie planu…" />;
  return <Summary details={summary.data} active={assignment.active} />;
}

function Summary({ details, active }: { details: CommercialSummary; active: boolean }) {
  return <div className="commercial-summary">
    <p><strong>{details.active_plan ? planCopy[details.active_plan] : "Brak aktywnego planu"}</strong> · {active ? "uczeń aktywny" : "uczeń nieaktywny"}</p>
    {details.package ? <p>Pakiet ważny do {details.package.valid_through}. Dostępne lekcje: {details.package.token_balance.available}, zarezerwowane: {details.package.token_balance.reserved}.</p> : null}
    {details.contract ? <p>Umowa {details.contract.start_on}–{details.contract.end_on}. Stan: {contractStatusCopy[details.contract.status]}. Cena: {formatMoney(details.contract.price_minor, details.contract.currency)} za lekcję. Pozostałe przełożenia w miesiącu: {details.contract.remaining_monthly_reschedules}. Bezpłatne odwołania: {details.contract.remaining_free_cancellations}.</p> : null}
    {details.active_plan ? null : <EmptyState>Wybierz plan w zakładce Plan, aby zacząć rozliczać lekcje.</EmptyState>}
    <p>Oczekujące płatności: {details.payments.pending}. Celowo nieopłacone: {details.payments.intentionally_unpaid}. Zaległe: {details.payments.overdue}. Nadpłata: {formatMoney(details.payments.credit_minor, details.payments.currency)}.</p>
  </div>;
}
