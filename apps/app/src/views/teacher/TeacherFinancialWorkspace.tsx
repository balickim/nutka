// Renders teacher action queues for ad hoc settlements, unpaid charges, overdue charges, refunds, and corrections.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { correctCharge, fetchFinancialWork, recordChargePayment, recordChargeRefund, recordSettlement } from "../../api/commercial";
import { chargeStateCopy, polishFinancialEntry } from "../../api/copy";
import type { Charge, FinancialEntry, UnresolvedWorkResponse } from "../../api/contracts";
import { ApiFeedback, EmptyState } from "../../components/ScheduleBits";
import { unresolvedWorkQuery, useCorrectionMutation, useSettlementMutation } from "../../query/commercial";
import { queryKeys } from "../../query/keys";
import { formatScheduleInstant } from "../../time/schedule";

export function TeacherFinancialWorkspace({ accountId }: { accountId: string }) {
  const unresolved = useQuery(unresolvedWorkQuery(accountId));
  const financial = useQuery(teacherFinancialWorkQuery(accountId));
  const [busy, setBusy] = useState<string | null>(null);
  const settlement = useSettlementActions(setBusy);
  const correction = useCorrectionActions(setBusy);
  const actions = { settleLesson: settlement.settleLesson, payCharge: settlement.payCharge, refund: correction.refund, correct: correction.correct };
  return <TeacherFinancialContent data={unresolved.data} entries={financial.data?.entries ?? []} credits={financial.data?.credits ?? []} busy={busy} actions={actions} error={[unresolved.error, financial.error, settlement.error, correction.error].find(Boolean)} onRetry={() => { void unresolved.refetch(); void financial.refetch(); }} />;
}

function teacherFinancialWorkQuery(accountId: string) {
  return { queryKey: queryKeys.financialWork("teacher", accountId), queryFn: ({ signal }: { signal: AbortSignal }) => fetchFinancialWork("teacher", undefined, signal) };
}

function useSettlementActions(setBusy: (value: string | null) => void) {
  const mutation = useSettlementMutation();
  const settleLesson = async (lesson: string, assignment: string, state: "paid" | "intentionally_unpaid") => {
    setBusy(lesson);
    try { await mutation.mutateAsync({ assignmentId: assignment, write: () => recordSettlement(lesson, { settlement: state }) }); }
    finally { setBusy(null); }
  };
  const payCharge = async (charge: Charge, state: "paid" | "intentionally_unpaid") => {
    setBusy(charge.id);
    try { await mutation.mutateAsync({ assignmentId: charge.assignment, write: () => recordChargePayment(charge.id, { settlement: state }) }); }
    finally { setBusy(null); }
  };
  return { settleLesson, payCharge, error: mutation.error };
}

function useCorrectionActions(setBusy: (value: string | null) => void) {
  const mutation = useCorrectionMutation();
  const refund = async (charge: Charge, amountMinor: number, reason: string) => {
    setBusy(charge.id);
    try { await mutation.mutateAsync({ assignmentId: charge.assignment, write: () => recordChargeRefund(charge.id, { amount_minor: amountMinor, reason }) }); }
    finally { setBusy(null); }
  };
  const correct = async (charge: Charge, reason: string) => {
    setBusy(charge.id);
    try { await mutation.mutateAsync({ assignmentId: charge.assignment, write: () => correctCharge(charge.id, { reason }) }); }
    finally { setBusy(null); }
  };
  return { refund, correct, error: mutation.error };
}

type FinancialActions = {
  settleLesson: (lesson: string, assignment: string, state: "paid" | "intentionally_unpaid") => Promise<void>;
  payCharge: (charge: Charge, state: "paid" | "intentionally_unpaid") => Promise<void>;
  refund: (charge: Charge, amountMinor: number, reason: string) => Promise<void>;
  correct: (charge: Charge, reason: string) => Promise<void>;
};

function TeacherFinancialContent({ data, entries, credits, busy, actions, error, onRetry }: { data?: UnresolvedWorkResponse; entries: FinancialEntry[]; credits: FinancialEntry[]; busy: string | null; actions: FinancialActions; error: unknown; onRetry: () => void }) {
  const count = unresolvedCount(data);
  return <section className="panel-section"><div className="section-heading"><h2>Do rozliczenia</h2><span className="counter"><strong>{count}</strong><span>spraw</span></span></div>
    <ApiFeedback error={error} onRetry={onRetry} />
    <UnresolvedQueue data={data} count={count} busy={busy} actions={actions} />
    <FinancialHistory entries={entries} credits={credits} />
  </section>;
}

function unresolvedCount(data?: UnresolvedWorkResponse): number {
  return data ? data.awaiting_outcome.length + data.pending_settlements.length + data.unpaid_charges.length + data.overdue_charges.length : 0;
}

function UnresolvedQueue({ data, count, busy, actions }: { data?: UnresolvedWorkResponse; count: number; busy: string | null; actions: FinancialActions }) {
  if (!data) return <p className="loading-state">Ładowanie kolejki…</p>;
  if (count === 0) return <EmptyState>Brak spraw wymagających działania.</EmptyState>;
  const charges = uniqueCharges([...data.unpaid_charges, ...data.overdue_charges]);
  return <div className="work-grid">
    {data.awaiting_outcome.length ? <p className="supporting-copy">Wyników oczekuje: {data.awaiting_outcome.length}. Formularze znajdują się przy zakończonych lekcjach.</p> : null}
    {data.pending_settlements.map((item) => <SettlementCard key={item.lesson} item={item} busy={busy === item.lesson} onSave={(state) => void actions.settleLesson(item.lesson, item.assignment, state)} />)}
    {charges.map((charge) => <ChargeCard key={charge.id} charge={charge} busy={busy === charge.id} onPay={(state) => void actions.payCharge(charge, state)} onRefund={(amount, reason) => void actions.refund(charge, amount, reason)} onCorrect={(reason) => void actions.correct(charge, reason)} />)}
  </div>;
}

function SettlementCard({ item, busy, onSave }: { item: { lesson: string; assignment: string; amount_minor: number; currency: string }; busy: boolean; onSave: (state: "paid" | "intentionally_unpaid") => void }) {
  const [state, setState] = useState<"paid" | "intentionally_unpaid">("paid");
  return <article className="assignment-card"><h3>Rozliczenie pojedynczej lekcji</h3><p>{(item.amount_minor / 100).toFixed(2)} {item.currency}</p><label htmlFor={`settlement-${item.lesson}`}>Stan</label><select id={`settlement-${item.lesson}`} value={state} onChange={(event) => setState(event.target.value as typeof state)}><option value="paid">Opłacona</option><option value="intentionally_unpaid">Celowo nieopłacona</option></select><p className="supporting-copy">Domyślny wybór nie zapisuje się automatycznie.</p><button className="primary-button" disabled={busy} onClick={() => onSave(state)}>Zapisz rozliczenie</button></article>;
}

function ChargeCard({ charge, busy, onPay, onRefund, onCorrect }: { charge: Charge; busy: boolean; onPay: (state: "paid" | "intentionally_unpaid") => void; onRefund: (amount: number, reason: string) => void; onCorrect: (reason: string) => void }) {
  const [state, setState] = useState<"paid" | "intentionally_unpaid">("paid");
  const [refund, setRefund] = useState("0");
  const [reason, setReason] = useState("");
  return <article className="assignment-card"><h3>{charge.overdue ? "Zaległa należność" : "Należność"}</h3><p>{(charge.current_amount_minor / 100).toFixed(2)} {charge.currency} · {chargeStateCopy[charge.derived_state] ?? "Nieznany stan"}</p><select aria-label="Stan należności" value={state} onChange={(event) => setState(event.target.value as typeof state)}><option value="paid">Opłacona</option><option value="intentionally_unpaid">Nieopłacona</option></select><button className="secondary-button" disabled={busy} onClick={() => onPay(state)}>Zapisz płatność</button><details><summary>Zwrot lub korekta</summary><label>Kwota zwrotu w groszach<input type="number" min="0" value={refund} onChange={(event) => setRefund(event.target.value)} /></label><label>Powód<input value={reason} onChange={(event) => setReason(event.target.value)} /></label><div className="row-actions"><button className="text-button" disabled={busy || !reason.trim()} onClick={() => onRefund(Number(refund), reason)}>Zapisz zwrot</button><button className="text-button" disabled={busy || !reason.trim()} onClick={() => onCorrect(reason)}>Zapisz korektę</button></div></details></article>;
}

function FinancialHistory({ entries, credits }: { entries: FinancialEntry[]; credits: FinancialEntry[] }) {
  const items = uniqueEntries([...credits, ...entries]);
  if (items.length === 0) return null;
  return <details className="financial-history"><summary>Kredyty, zwroty i historia rozliczeń</summary><ul className="history-list">{items.map((entry) => <li key={entry.id}><strong>{polishFinancialEntry(entry.entry_type)}</strong> · {(entry.amount_minor / 100).toFixed(2)} {entry.currency} · {formatScheduleInstant(entry.event_at)}</li>)}</ul></details>;
}

function uniqueCharges(charges: Charge[]): Charge[] {
  return Array.from(new Map(charges.map((charge) => [charge.id, charge])).values());
}

function uniqueEntries(entries: FinancialEntry[]): FinancialEntry[] {
  return Array.from(new Map(entries.map((entry) => [entry.id, entry])).values()).sort((left, right) => right.event_at.localeCompare(left.event_at));
}
