// Shows the plan of one assignment: summary, contract notice, flexible booking, history, and package token details.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { contractStatusCopy, planCopy, polishEvent, polishTokenState } from "../../../api/copy";
import { submitContractNotice } from "../../../api/commercial";
import type { Assignment, CommercialSummary, HistoryEvent, Policy, Slot } from "../../../api/contracts";
import { ApiFeedback } from "../../../components/api-feedback";
import { ConfirmDialog } from "../../../components/confirm-dialog";
import { Skeleton } from "../../../components/skeleton";
import { useToast } from "../../../components/toast";
import { formatMoney } from "../../../money";
import { commercialSummaryQuery, historyQuery, usePlanMutation } from "../../../query/commercial";
import { formatLocalDate, formatScheduleInstant } from "../../../time/schedule";
import { LearnerBooking } from "./booking";
import { paymentFacts } from "./payment-facts";

export function PlanCard({ accountId, assignment, slots, policy }: { accountId: string; assignment: Assignment; slots: Slot[]; policy: Policy }) {
  const summary = useQuery(commercialSummaryQuery("learner", accountId, assignment.id));
  const history = useQuery(historyQuery("learner", accountId, assignment.id));
  const plan = usePlanMutation();
  const { notify } = useToast();
  const [noticeOpen, setNoticeOpen] = useState(false);
  const contract = summary.data?.contract;
  async function notice() {
    if (!contract) return;
    await plan.mutateAsync({ assignmentId: assignment.id, write: () => submitContractNotice("learner", contract.id) });
    notify("Wypowiedzenie złożone.");
    setNoticeOpen(false);
  }
  return <article className="assignment-card">
    <ApiFeedback error={summary.error || history.error || plan.error} />
    <PlanSummary details={summary.data} pending={summary.isPending} onNotice={() => setNoticeOpen(true)} />
    <ConfirmDialog open={noticeOpen} title="Wypowiedzenie umowy" consequence={contract ? `Umowa zakończy się ${formatLocalDate(contract.end_on)}. Lekcje po tej dacie znikną z kalendarza.` : ""} confirmLabel="Złóż wypowiedzenie" danger busy={plan.isPending} onConfirm={() => void notice()} onCancel={() => setNoticeOpen(false)} />
    {summary.data?.active_plan === "regular_contract" ? <p className="supporting-copy">Stałe terminy wynikają z umowy. Elastyczna rezerwacja jest wyłączona.</p> : <LearnerBooking assignment={assignment} slots={slots} policy={policy} />}
    <LearnerHistory items={history.data?.items ?? []} />
    <TokenDetails details={summary.data} />
  </article>;
}

function PlanSummary({ details, pending, onNotice }: { details?: CommercialSummary; pending: boolean; onNotice: () => void }) {
  if (pending || !details) return <Skeleton lines={3} label="Ładowanie planu…" />;
  const { package: lessonPackage, contract } = details;
  return <div className="commercial-summary">
    <strong>{details.active_plan ? planCopy[details.active_plan] : "Brak aktywnego planu"}</strong>
    {lessonPackage ? <p>Pakiet ważny do {formatLocalDate(lessonPackage.valid_through)}. Wolne lekcje: {lessonPackage.token_balance.available}, zarezerwowane: {lessonPackage.token_balance.reserved}.</p> : null}
    {contract ? <p>Umowa od {formatLocalDate(contract.start_on)} do {formatLocalDate(contract.end_on)}. Stan: {contractStatusCopy[contract.status]}. Cena: {formatMoney(contract.price_minor, contract.currency)} za lekcję. Przełożenia w tym miesiącu: {contract.remaining_monthly_reschedules}. Bezpłatne odwołania: {contract.remaining_free_cancellations}.</p> : null}
    {paymentFacts(details.payments).map((fact) => <p key={fact}>{fact}</p>)}
    {contract ? <button className="text-button danger-button" onClick={onNotice}>Złóż wypowiedzenie</button> : null}
  </div>;
}

function LearnerHistory({ items }: { items: HistoryEvent[] }) {
  return <details><summary>Historia</summary>{items.length ? <ul className="history-list">{items.map((event) => <li key={event.id}><strong>{polishEvent(event.event_type)}</strong> · {formatScheduleInstant(event.event_at)}{event.corrects_event ? " · korekta" : ""}</li>)}</ul> : <p className="supporting-copy">Brak zdarzeń.</p>}</details>;
}

function TokenDetails({ details }: { details?: CommercialSummary }) {
  if (!details?.package) return null;
  return <details><summary>Lekcje w pakiecie</summary><p className="supporting-copy">{Object.entries(details.package.token_balance).map(([state, count]) => `${polishTokenState(state)}: ${count}`).join(" · ")}</p></details>;
}
