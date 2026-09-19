// Lists the contract occurrences and the monthly forecast or charge for one contract.

import { useQuery } from "@tanstack/react-query";

import { fetchContractMonths, fetchContractSeries } from "../../../../api/commercial";
import { billingOutcomeCopy, chargeStateCopy, scheduleStateCopy } from "../../../../api/copy";
import type { ContractMonth, ContractOccurrence, RegularContract } from "../../../../api/contracts";
import { ApiFeedback } from "../../../../components/api-feedback";
import { formatMoney } from "../../../../money";
import { queryKeys } from "../../../../query/keys";
import { formatScheduleInstant } from "../../../../time/schedule";

export function ContractSeries({ assignmentId, contract }: { assignmentId: string; contract: RegularContract }) {
  const series = useQuery(seriesQuery(assignmentId, contract.id));
  const months = useQuery(monthsQuery(assignmentId, contract.id));
  return <>
    <ApiFeedback error={firstError(series.error, months.error)} />
    <SeriesList values={series.data} />
    <MonthList months={monthList(months.data)} contract={contract} />
  </>;
}

function firstError(left: unknown, right: unknown): unknown {
  return left || right;
}

function monthList(value?: { months: ContractMonth[] }): ContractMonth[] {
  return value ? value.months : [];
}

function SeriesList({ values }: { values?: { near_term: ContractOccurrence[]; later: ContractOccurrence[] } }) {
  const nearTerm = values ? values.near_term : [];
  const later = values ? values.later : [];
  return <details><summary>Seria lekcji</summary><h4>Najbliższe</h4><OccurrenceList values={nearTerm} /><h4>Dalsze</h4><OccurrenceList values={later} /></details>;
}

function MonthList({ months, contract }: { months: ContractMonth[]; contract: RegularContract }) {
  return <details><summary>Prognozy i należności miesięczne</summary><ul className="history-list">{months.map((month) => <li key={month.id}><strong>{month.month}</strong> · {monthAmount(month, contract)} · {monthState(month)}</li>)}</ul></details>;
}

function seriesQuery(assignmentId: string, contractId: string) {
  return { queryKey: queryKeys.contractSeries("teacher", assignmentId, contractId), queryFn: ({ signal }: { signal: AbortSignal }) => fetchContractSeries("teacher", contractId, signal) };
}

function monthsQuery(assignmentId: string, contractId: string) {
  return { queryKey: queryKeys.contractMonths("teacher", assignmentId, contractId), queryFn: ({ signal }: { signal: AbortSignal }) => fetchContractMonths("teacher", contractId, signal) };
}

function monthAmount(month: ContractMonth, contract: RegularContract): string {
  return formatMoney(month.current_amount_minor ?? month.forecast_amount_minor, month.currency ?? contract.currency);
}

function monthState(month: ContractMonth): string {
  if (month.forecast) return "prognoza";
  return chargeStateCopy[month.derived_state ?? month.settlement_state ?? "pending"] ?? "nieznany stan";
}

function OccurrenceList({ values }: { values: ContractOccurrence[] }) {
  if (!values.length) return <p className="supporting-copy">Brak lekcji.</p>;
  return <ul className="history-list">{values.map((occurrence) => <li key={occurrence.id}>{formatScheduleInstant(occurrence.start_at)} · {scheduleStateCopy[occurrence.schedule_state]} · {billingOutcomeCopy[occurrence.billing_outcome] ?? "Nieznany skutek rozliczenia"}</li>)}</ul>;
}
