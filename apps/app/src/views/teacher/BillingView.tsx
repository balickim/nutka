// Presents the teacher's unresolved financial work as one queue, with the learner name and cause on every entry.

import { useQuery } from "@tanstack/react-query";

import { fetchFinancialWork } from "../../api/commercial";
import { polishFinancialEntry } from "../../api/copy";
import type { FinancialEntry, UnresolvedWorkResponse } from "../../api/contracts";
import { ApiFeedback } from "../../components/ApiFeedback";
import { EmptyState } from "../../components/EmptyState";
import { Skeleton } from "../../components/Skeleton";
import { formatMoney } from "../../money";
import { unresolvedWorkQuery } from "../../query/commercial";
import { queryKeys } from "../../query/keys";
import { formatScheduleInstant } from "../../time/schedule";
import { ChargeCard, SettlementCard } from "./billing/QueueCards";
import { learnerNames } from "./roster";
import { TeacherShell } from "./TeacherShell";

export function BillingView() {
  return <TeacherShell lede="Rozlicz lekcje i należności, które czekają na Twoją decyzję.">
    {(context) => <BillingQueue accountId={context.accountId} names={learnerNames(context.calendar)} />}
  </TeacherShell>;
}

function BillingQueue({ accountId, names }: { accountId: string; names: ReadonlyMap<string, string> }) {
  const unresolved = useQuery(unresolvedWorkQuery(accountId));
  const financial = useQuery(financialWorkQuery(accountId));
  const error = [unresolved.error, financial.error].find(Boolean);
  const retry = () => { void unresolved.refetch(); void financial.refetch(); };
  return <section className="panel-section">
    <div className="section-heading"><h2>Do rozliczenia</h2><span className="counter"><strong>{unresolvedCount(unresolved.data)}</strong><span>spraw</span></span></div>
    <ApiFeedback error={error} onRetry={retry} />
    {error ? null : <QueueBody data={unresolved.data} pending={unresolved.isPending} names={names} />}
    <FinancialHistory entries={financial.data?.entries ?? []} credits={financial.data?.credits ?? []} />
  </section>;
}

function financialWorkQuery(accountId: string) {
  return { queryKey: queryKeys.financialWork("teacher", accountId), queryFn: ({ signal }: { signal: AbortSignal }) => fetchFinancialWork("teacher", undefined, signal) };
}

export function unresolvedCount(data?: UnresolvedWorkResponse): number {
  return data ? data.awaiting_outcome.length + data.pending_settlements.length + uniqueCharges(data).length : 0;
}

export function uniqueCharges(data: UnresolvedWorkResponse) {
  return Array.from(new Map([...data.unpaid_charges, ...data.overdue_charges].map((charge) => [charge.id, charge])).values());
}

function QueueBody({ data, pending, names }: { data?: UnresolvedWorkResponse; pending: boolean; names: ReadonlyMap<string, string> }) {
  if (pending || !data) return <Skeleton lines={4} label="Ładowanie kolejki…" />;
  if (unresolvedCount(data) === 0) return <EmptyState action={<a className="btn btn-ghost btn-sm" href="/teachers/students">Otwórz listę uczniów</a>}>Wszystko rozliczone. Nic nie czeka na Twoją decyzję.</EmptyState>;
  return <div className="work-grid">
    {data.awaiting_outcome.length ? <p className="supporting-copy">Wyników oczekuje: {data.awaiting_outcome.length}. Zapisz je na ekranie Dziś.</p> : null}
    {data.pending_settlements.map((item) => <SettlementCard key={item.lesson} item={item} learner={names.get(item.assignment)} />)}
    {uniqueCharges(data).map((charge) => <ChargeCard key={charge.id} charge={charge} learner={names.get(charge.assignment)} />)}
  </div>;
}

function FinancialHistory({ entries, credits }: { entries: FinancialEntry[]; credits: FinancialEntry[] }) {
  const items = Array.from(new Map([...credits, ...entries].map((entry) => [entry.id, entry])).values()).sort((left, right) => right.event_at.localeCompare(left.event_at));
  if (items.length === 0) return null;
  return <details className="financial-history"><summary>Kredyty, zwroty i historia rozliczeń</summary><ul className="history-list">{items.map((entry) => <li key={entry.id}><strong>{polishFinancialEntry(entry.entry_type)}</strong> · {formatMoney(entry.amount_minor, entry.currency)} · {formatScheduleInstant(entry.event_at)}</li>)}</ul></details>;
}
