// Renders assignment-scoped package, contract, teacher booking, transition, series, correction, and history controls.

import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";

import {
  activateContract,
  amendContractPrice,
  bookFlexibleLesson,
  changeContractSchedule,
  closePackage,
  correctContract,
  correctPackage,
  endContractEarly,
  fetchContracts,
  fetchContractSeries,
  fetchContractMonths,
  fetchPackages,
  purchasePackage,
  renewContract,
  submitContractNotice,
  updateAssignment,
} from "../../api/commercial";
import { billingOutcomeCopy, chargeStateCopy, planCopy, polishEvent, polishTokenState, scheduleStateCopy } from "../../api/copy";
import type { Assignment, CommercialSummary, ContractOccurrence, Package, Policy, RegularContract } from "../../api/contracts";
import { assignmentDisplayName } from "../../api/scheduling";
import { ApiFeedback, EmptyState } from "../../components/ScheduleBits";
import { commercialSummaryQuery, historyQuery, useBookingMutation, useCorrectionMutation, usePlanMutation } from "../../query/commercial";
import { queryKeys } from "../../query/keys";
import { useAssignmentWrite } from "../../query/scheduling";
import { formatScheduleInstant, localInputToUtc, weekdayLabels } from "../../time/schedule";

export function TeacherCommercialWorkspace({ accountId, assignments, policy }: { accountId: string; assignments: Assignment[]; policy: Policy }) {
  if (assignments.length === 0) return <section className="panel-section"><h2>Uczniowie</h2><EmptyState>Nie masz przypisanych uczniów.</EmptyState></section>;
  return <section className="panel-section"><h2>Plany uczniów</h2><div className="assignment-list">{assignments.map((assignment) => <CommercialCard key={assignment.id} accountId={accountId} assignment={assignment} policy={policy} />)}</div></section>;
}

function CommercialCard({ accountId, assignment, policy }: { accountId: string; assignment: Assignment; policy: Policy }) {
  const data = useCommercialCardData(accountId, assignment.id);
  const assignmentWrite = useAssignmentWrite();
  const error = [data.error, assignmentWrite.error].find(Boolean);
  return <article className="assignment-card"><div className="assignment-heading"><div><p className="eyebrow">Uczeń</p><h3>{assignmentDisplayName(assignment, "teacher")}</h3></div><span className="duration-badge">45 min</span></div>
    <ApiFeedback error={error} />
    <AssignmentState assignment={assignment} activePlan={data.activePlan} onToggle={() => void assignmentWrite.mutateAsync({ assignmentId: assignment.id, write: () => updateAssignment(assignment.id, { active: !assignment.active }) })} />
    <div className="commercial-columns"><PackageControls assignment={assignment} packages={data.packages} policy={policy} /><ContractControls assignment={assignment} contracts={data.contracts} policy={policy} /><TeacherBooking assignment={assignment} policy={policy} /></div>
    <TeacherHistory items={data.history} />
  </article>;
}

function useCommercialCardData(accountId: string, assignmentId: string) {
  const summary = useQuery(commercialSummaryQuery("teacher", accountId, assignmentId));
  const packages = useQuery(packageListQuery(accountId, assignmentId));
  const contracts = useQuery(contractListQuery(accountId, assignmentId));
  const history = useQuery(historyQuery("teacher", accountId, assignmentId));
  return {
    activePlan: activePlanValue(summary.data),
    packages: packageItems(packages.data),
    contracts: contractItems(contracts.data),
    history: historyItems(history.data),
    error: [summary.error, packages.error, contracts.error, history.error].find(Boolean),
  };
}

type TeacherHistoryItem = { id: string; event_type: Parameters<typeof polishEvent>[0]; event_at: string; corrects_event?: string; internal_note?: string };

function activePlanValue(summary?: CommercialSummary) {
  return summary ? summary.active_plan : undefined;
}

function packageItems(value?: { packages: Package[] }): Package[] {
  return value ? value.packages : [];
}

function contractItems(value?: { contracts: RegularContract[] }): RegularContract[] {
  return value ? value.contracts : [];
}

function historyItems(value?: { items: TeacherHistoryItem[] }): TeacherHistoryItem[] {
  return value ? value.items : [];
}

function packageListQuery(accountId: string, assignmentId: string) {
  return { queryKey: [...queryKeys.commercialSummary("teacher", accountId, assignmentId), "packages"], queryFn: ({ signal }: { signal: AbortSignal }) => fetchPackages("teacher", assignmentId, signal) };
}

function contractListQuery(accountId: string, assignmentId: string) {
  return { queryKey: [...queryKeys.commercialSummary("teacher", accountId, assignmentId), "contracts"], queryFn: ({ signal }: { signal: AbortSignal }) => fetchContracts("teacher", assignmentId, signal) };
}

function AssignmentState({ assignment, activePlan, onToggle }: { assignment: Assignment; activePlan?: keyof typeof planCopy | null; onToggle: () => void }) {
  return <><p>{activePlan ? planCopy[activePlan] : "Brak aktywnego planu"} · {assignment.active ? "aktywne przypisanie" : "nieaktywne przypisanie"}</p><button className="text-button" onClick={onToggle}>{assignment.active ? "Dezaktywuj" : "Aktywuj"}</button></>;
}

function TeacherHistory({ items }: { items: TeacherHistoryItem[] }) {
  return <details><summary>Pełna historia</summary>{items.length ? <ul className="history-list">{items.map((event) => <li key={event.id}><strong>{polishEvent(event.event_type)}</strong> · {formatScheduleInstant(event.event_at)}{event.corrects_event ? ` · koryguje ${event.corrects_event}` : ""}{event.internal_note ? ` · ${event.internal_note}` : ""}</li>)}</ul> : <p className="supporting-copy">Brak zdarzeń.</p>}</details>;
}

function PackageControls({ assignment, packages, policy }: { assignment: Assignment; packages: Package[]; policy: Policy }) {
  const mutation = usePlanMutation();
  const correction = useCorrectionMutation();
  const [purchaseDate, setPurchaseDate] = useState("");
  const [convert, setConvert] = useState("");
  const [reason, setReason] = useState("");
  const [refundMinor, setRefundMinor] = useState("0");
  const [refundNote, setRefundNote] = useState("");
  const current = packages.find((item) => item.status === "open");
  const canRenew = current && current.tokens.every((token) => token.state !== "available");
  async function purchase() {
    await mutation.mutateAsync({ assignmentId: assignment.id, write: () => purchasePackage(assignment.id, { ...(purchaseDate ? { purchased_on: purchaseDate } : {}), ...(ids(convert).length ? { convert_lesson_ids: ids(convert) } : {}) }) });
  }
  async function close() {
    if (!current || !reason.trim()) return;
    const amount = Number(refundMinor);
    await mutation.mutateAsync({ assignmentId: assignment.id, write: () => closePackage(current.id, { reason, ...(amount > 0 ? { refund: { amount_minor: amount, currency: "PLN", ...(refundNote.trim() ? { note: refundNote.trim() } : {}) } } : {}) }) });
  }
  async function correct(token: Package["tokens"][number]) {
    const event = window.prompt("Identyfikator zdarzenia do korekty");
    if (!current || !event || !reason.trim()) return;
    await correction.mutateAsync({ assignmentId: assignment.id, write: () => correctPackage(current.id, { token_id: token.id, target: "available", corrects_event: event, reason }) });
  }
  return <section className="subpanel"><h3>Pakiet {policy.package_token_count} lekcji</h3><ApiFeedback error={mutation.error || correction.error} />{current ? <><p>Ważny do {current.valid_through}. Cena {(current.price_minor / 100).toFixed(0)} {current.currency}.</p><ul>{current.tokens.map((token) => <li key={token.id}>#{token.ordinal}: {polishTokenState(token.state)} {token.state !== "available" ? <button className="text-button" onClick={() => void correct(token)}>Koryguj</button> : null}</li>)}</ul>{canRenew ? <button className="secondary-button" onClick={() => void purchase()}>Oznacz odnowienie pakietu</button> : null}<label>Powód zamknięcia<input value={reason} onChange={(event) => setReason(event.target.value)} /></label><label>Opcjonalny zwrot w groszach<input type="number" min="0" value={refundMinor} onChange={(event) => setRefundMinor(event.target.value)} /></label><label>Notatka do zwrotu<input value={refundNote} onChange={(event) => setRefundNote(event.target.value)} /></label><button className="text-button danger-button" disabled={!reason.trim()} onClick={() => void close()}>Zamknij pakiet</button></> : <><p>{(policy.package_price_minor / 100).toFixed(0)} {policy.currency}, ważny {policy.package_validity_days} dni.</p><label>Data zakupu, opcjonalnie<input type="date" value={purchaseDate} onChange={(event) => setPurchaseDate(event.target.value)} /></label><label>Identyfikatory lekcji ad hoc do konwersji<input value={convert} onChange={(event) => setConvert(event.target.value)} placeholder="id1, id2" /></label><button className="secondary-button" onClick={() => void purchase()}>Oznacz zakup pakietu</button></>}</section>;
}

function ContractControls({ assignment, contracts, policy }: { assignment: Assignment; contracts: RegularContract[]; policy: Policy }) {
  const mutation = usePlanMutation();
  const active = contracts.find((contract) => contract.status !== "ended");
  const [startOn, setStartOn] = useState("");
  const [weekday, setWeekday] = useState("1");
  const [startTime, setStartTime] = useState("17:00");
  const [convert, setConvert] = useState("");
  const [backdateReason, setBackdateReason] = useState("");
  const [pastOutcomes, setPastOutcomes] = useState("");
  async function activate(event: FormEvent) {
    event.preventDefault();
    await mutation.mutateAsync({ assignmentId: assignment.id, write: () => activateContract(assignment.id, { start_on: startOn, weekday: Number(weekday), start_time: startTime, ...(ids(convert).length ? { convert_lesson_ids: ids(convert) } : {}), ...(backdateReason.trim() ? { backdate_reason: backdateReason.trim() } : {}), ...(outcomeMap(pastOutcomes) ? { past_outcomes: outcomeMap(pastOutcomes) } : {}) }) });
  }
  return <section className="subpanel"><h3>Umowa tygodniowa</h3><ApiFeedback error={mutation.error} />{active ? <ContractActions assignment={assignment.id} contract={active} /> : <form onSubmit={(event) => void activate(event)}><p>{(policy.regular_lesson_price_minor / 100).toFixed(0)} {policy.currency} za {policy.lesson_duration_minutes} minut. Koniec: {policy.contract_end_day}.{policy.contract_end_month}. Limit miesięcznych przełożeń: {policy.contract_monthly_reschedules}. Bezpłatne odwołania: {policy.contract_free_cancellations}. Termin zastępczy: {policy.contract_replacement_deadline_days} dni.</p><label>Data początku<input type="date" value={startOn} onChange={(event) => setStartOn(event.target.value)} required /></label><label>Dzień tygodnia<select value={weekday} onChange={(event) => setWeekday(event.target.value)}>{weekdayLabels.map((label, index) => <option key={label} value={index}>{label}</option>)}</select></label><label>Godzina<input type="time" step={policy.start_grid_minutes * 60} value={startTime} onChange={(event) => setStartTime(event.target.value)} required /></label><label>Lekcje ad hoc do konwersji<input value={convert} onChange={(event) => setConvert(event.target.value)} /></label><label>Powód aktywacji wstecznej<input value={backdateReason} onChange={(event) => setBackdateReason(event.target.value)} /></label><label>Wyniki przeszłych lekcji, jedna linia: data=wynik<textarea value={pastOutcomes} onChange={(event) => setPastOutcomes(event.target.value)} placeholder="2030-09-02=odbyta" /></label><button className="secondary-button" type="submit">Aktywuj umowę</button></form>}</section>;
}

function ContractActions({ assignment, contract }: { assignment: string; contract: RegularContract }) {
  const series = useQuery({ queryKey: queryKeys.contractSeries("teacher", assignment, contract.id), queryFn: ({ signal }) => fetchContractSeries("teacher", contract.id, signal) });
  const months = useQuery({ queryKey: [...queryKeys.contractSeries("teacher", assignment, contract.id), "months"], queryFn: ({ signal }) => fetchContractMonths("teacher", contract.id, signal) });
  const mutation = usePlanMutation();
  const correction = useCorrectionMutation();
  const [date, setDate] = useState(contract.start_on);
  const [time, setTime] = useState(contract.start_time);
  const [weekday, setWeekday] = useState(String(contract.weekday));
  const [reason, setReason] = useState("");
  const [price, setPrice] = useState(String(contract.price_minor));
  const [eventId, setEventId] = useState("");
  const write = (operation: () => Promise<unknown>) => mutation.mutateAsync({ assignmentId: assignment, write: operation });
  return <div><ApiFeedback error={series.error || months.error} /><p>{contract.start_on}–{contract.effective_end_on || contract.end_on}, {contract.start_time}. Zmiany: {contract.remaining_monthly_reschedules}, odwołania: {contract.remaining_free_cancellations}.</p><div className="compact-fields"><input aria-label="Data obowiązywania" type="date" value={date} onChange={(event) => setDate(event.target.value)} /><input aria-label="Godzina umowy" type="time" value={time} onChange={(event) => setTime(event.target.value)} /><select aria-label="Dzień umowy" value={weekday} onChange={(event) => setWeekday(event.target.value)}>{weekdayLabels.map((label, day) => <option key={label} value={day}>{label}</option>)}</select></div><div className="row-actions"><button className="text-button" onClick={() => void write(() => changeContractSchedule(contract.id, { effective_on: date, weekday: Number(weekday), start_time: time }))}>Zmień stały termin</button><button className="text-button" onClick={() => void write(() => submitContractNotice("teacher", contract.id))}>Wypowiedz</button><button className="text-button" onClick={() => void write(() => renewContract(contract.id, { start_on: date }))}>Odnów</button></div><label>Powód wcześniejszego końca lub korekty<input value={reason} onChange={(event) => setReason(event.target.value)} /></label><label>Id zdarzenia<input value={eventId} onChange={(event) => setEventId(event.target.value)} /></label><div className="row-actions"><button className="text-button danger-button" disabled={!reason} onClick={() => void write(() => endContractEarly(contract.id, { end_on: date, reason }))}>Zakończ wcześniej</button><button className="text-button" disabled={!reason || !eventId} onClick={() => void correction.mutateAsync({ assignmentId: assignment, write: () => correctContract(contract.id, { event_id: eventId, reason }) })}>Koryguj</button></div><label>Nowa cena w groszach<input type="number" value={price} onChange={(event) => setPrice(event.target.value)} /></label><button className="text-button" onClick={() => void write(() => amendContractPrice(contract.id, { effective_on: date, price_minor: Number(price), currency: "PLN" }))}>Zapisz przyszłą cenę</button><details><summary>Seria: najbliższe {series.data?.near_term.length ?? 0}, dalsze {series.data?.later.length ?? 0}</summary><h4>Lekcje w horyzoncie</h4><OccurrenceList values={series.data?.near_term ?? []} /><h4>Dalsze stałe rezerwacje</h4><OccurrenceList values={series.data?.later ?? []} /></details><details><summary>Prognozy i należności miesięczne</summary><ul className="history-list">{(months.data?.months ?? []).map((month) => <li key={month.id}><strong>{month.month}</strong> · {((month.current_amount_minor ?? month.forecast_amount_minor) / 100).toFixed(2)} {month.currency ?? contract.currency} · {month.forecast ? "prognoza" : chargeStateCopy[month.derived_state ?? month.settlement_state ?? "pending"] ?? "nieznany stan"}</li>)}</ul></details></div>;
}

function OccurrenceList({ values }: { values: ContractOccurrence[] }) {
  if (!values.length) return <p className="supporting-copy">Brak lekcji.</p>;
  return <ul className="history-list">{values.map((occurrence) => <li key={occurrence.id}>{formatScheduleInstant(occurrence.start_at)} · {scheduleStateCopy[occurrence.schedule_state]} · {billingOutcomeCopy[occurrence.billing_outcome] ?? "Nieznany skutek rozliczenia"}</li>)}</ul>;
}

function TeacherBooking({ assignment, policy }: { assignment: Assignment; policy: Policy }) {
  const mutation = useBookingMutation();
  const [start, setStart] = useState("");
  const [confirm, setConfirm] = useState(false);
  const shortNotice = start ? Date.parse(localInputToUtc(start)) - Date.now() < policy.learner_booking_minimum_hours * 60 * 60 * 1000 : false;
  return <section className="subpanel"><h3>Zarezerwuj za ucznia</h3><ApiFeedback error={mutation.error} /><p>Termin musi zaczynać się w ciągu {policy.booking_horizon_days} dni, na siatce co {policy.start_grid_minutes} minut. Bufor uczestnika wynosi {policy.participant_buffer_minutes} minut.</p><label>Termin<input type="datetime-local" step={policy.start_grid_minutes * 60} value={start} onChange={(event) => { setStart(event.target.value); setConfirm(false); }} /></label>{shortNotice ? <label className="check-label"><input type="checkbox" checked={confirm} onChange={(event) => setConfirm(event.target.checked)} /> Potwierdzam termin krótszy niż {policy.learner_booking_minimum_hours} godz.</label> : null}<button className="secondary-button" disabled={!start || (shortNotice && !confirm)} onClick={() => void mutation.mutateAsync({ assignmentId: assignment.id, write: () => bookFlexibleLesson("teacher", assignment.id, { start_at: localInputToUtc(start), ...(confirm ? { confirm_short_notice: true } : {}) }) })}>Zarezerwuj</button></section>;
}

function ids(value: string): string[] {
  return value.split(",").map((item) => item.trim()).filter(Boolean);
}

function outcomeMap(value: string): Record<string, string> | undefined {
  const labels: Record<string, string> = { odbyta: "completed", "odwołana": "cancelled", nieobecność: "no_show", zaplanowana: "scheduled" };
  const entries = value.split("\n").map((line) => line.split("=").map((item) => item.trim())).filter(([date, outcome]) => date && outcome).map(([date, outcome]) => [date, labels[outcome.toLowerCase()] ?? outcome]);
  return entries.length ? Object.fromEntries(entries) : undefined;
}
