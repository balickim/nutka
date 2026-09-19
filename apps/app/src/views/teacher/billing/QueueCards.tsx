// Renders one queue entry per unresolved settlement or charge, with its learner, amount, cause, and tier A action.

import { useState } from "react";

import { correctCharge, recordChargePayment, recordChargeRefund, recordSettlement } from "../../../api/commercial";
import { chargeStateCopy } from "../../../api/copy";
import type { Charge } from "../../../api/contracts";
import { ActionButton } from "../../../components/ActionButton";
import { AdvancedOperations } from "../../../components/AdvancedOperations";
import { ApiFeedback } from "../../../components/ApiFeedback";
import { useToast } from "../../../components/Toast";
import { formatMoney, parseMajor } from "../../../money";
import { useCorrectionMutation, useSettlementMutation } from "../../../query/commercial";
import { formatScheduleDate } from "../../../time/schedule";

type Settlement = { lesson: string; assignment: string; amount_minor: number; currency: string };
type SettlementState = "paid" | "intentionally_unpaid";

export function SettlementCard({ item, learner }: { item: Settlement; learner?: string }) {
  const mutation = useSettlementMutation();
  const { notify } = useToast();
  const [state, setState] = useState<SettlementState>("paid");
  async function save() {
    await mutation.mutateAsync({ assignmentId: item.assignment, write: () => recordSettlement(item.lesson, { settlement: state }) });
    notify(`Zapisano rozliczenie lekcji: ${formatMoney(item.amount_minor, item.currency)}.`);
  }
  return <article className="assignment-card">
    <p className="eyebrow">Pojedyncza lekcja</p>
    <h3>{learner ?? "Uczeń"}</h3>
    <p className="lesson-meta">{formatMoney(item.amount_minor, item.currency)} · czeka na rozliczenie</p>
    <ApiFeedback error={mutation.error} />
    <label htmlFor={`settlement-${item.lesson}`}>Stan</label>
    <select id={`settlement-${item.lesson}`} value={state} onChange={(event) => setState(event.target.value as SettlementState)}><option value="paid">Opłacona</option><option value="intentionally_unpaid">Celowo nieopłacona</option></select>
    <ActionButton variant="primary" busy={mutation.isPending} onClick={() => void save()}>Zapisz rozliczenie</ActionButton>
  </article>;
}

export function ChargeCard({ charge, learner }: { charge: Charge; learner?: string }) {
  const settlement = useSettlementMutation();
  const { notify } = useToast();
  const [state, setState] = useState<SettlementState>("paid");
  async function pay() {
    await settlement.mutateAsync({ assignmentId: charge.assignment, write: () => recordChargePayment(charge.id, { settlement: state }) });
    notify(`Zapisano płatność należności: ${formatMoney(charge.current_amount_minor, charge.currency)}.`);
  }
  return <article className="assignment-card">
    <div className="assignment-heading"><div><p className="eyebrow">{charge.overdue ? "Zaległa należność" : "Należność"}</p><h3>{learner ?? "Uczeń"}</h3></div><span className={`status-badge status-${charge.overdue ? "overdue" : "pending"}`}>{chargeStateCopy[charge.derived_state] ?? "Nieznany stan"}</span></div>
    <p className="lesson-meta">{formatMoney(charge.current_amount_minor, charge.currency)} · termin {formatScheduleDate(`${charge.due_on}T00:00:00Z`)}</p>
    <ApiFeedback error={settlement.error} />
    <label htmlFor={`charge-${charge.id}`}>Stan</label>
    <select id={`charge-${charge.id}`} value={state} onChange={(event) => setState(event.target.value as SettlementState)}><option value="paid">Opłacona</option><option value="intentionally_unpaid">Celowo nieopłacona</option></select>
    <ActionButton variant="primary" busy={settlement.isPending} onClick={() => void pay()}>Zapisz płatność</ActionButton>
    <ChargeCorrections charge={charge} />
  </article>;
}

function ChargeCorrections({ charge }: { charge: Charge }) {
  const mutation = useCorrectionMutation();
  const { notify } = useToast();
  const [refund, setRefund] = useState("0,00");
  const amount = parseMajor(refund);
  async function write(operation: () => Promise<unknown>, message: string, reason: string) {
    await mutation.mutateAsync({ assignmentId: charge.assignment, write: operation });
    notify(`${message} Powód: ${reason}`);
  }
  return <AdvancedOperations label="Zwrot lub korekta należności">
    {(reason) => <>
      <ApiFeedback error={mutation.error} />
      <label htmlFor={`refund-${charge.id}`}>Kwota zwrotu w {charge.currency}</label>
      <input id={`refund-${charge.id}`} inputMode="decimal" value={refund} onChange={(event) => setRefund(event.target.value)} aria-invalid={amount === null} />
      {amount === null ? <p className="field-error">Podaj kwotę, na przykład 25,00.</p> : null}
      <div className="row-actions">
        <ActionButton busy={mutation.isPending} disabled={!reason.trim() || amount === null || amount <= 0} onClick={() => void write(() => recordChargeRefund(charge.id, { amount_minor: amount ?? 0, reason }), "Zapisano zwrot.", reason)}>Zapisz zwrot</ActionButton>
        <ActionButton busy={mutation.isPending} disabled={!reason.trim()} onClick={() => void write(() => correctCharge(charge.id, { reason }), "Zapisano korektę.", reason)}>Zapisz korektę</ActionButton>
      </div>
    </>}
  </AdvancedOperations>;
}
