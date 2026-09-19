// Runs contract activation, slot change, notice, renewal, and price change as guided flows, with corrections disclosed.

import { useState } from "react";

import { activateContract } from "../../../api/commercial";
import type { Lesson, Policy, RegularContract } from "../../../api/contracts";
import { ApiFeedback } from "../../../components/api-feedback";
import { FlowPanel } from "../../../components/flow-panel";
import { PolicyHint } from "../../../components/policy-hint";
import { useToast } from "../../../components/toast";
import { formatMoney } from "../../../money";
import { usePlanMutation } from "../../../query/commercial";
import { formatScheduleDate, weekdayLabels } from "../../../time/schedule";
import { ContractActions } from "./contract-actions";
import { LessonPicker } from "./lesson-picker";
import { PastOutcomes } from "./past-outcomes";

export function ContractPanel({ accountId, assignmentId, contracts, adHocLessons, policy }: { accountId: string; assignmentId: string; contracts: RegularContract[]; adHocLessons: Lesson[]; policy: Policy }) {
  const active = contracts.find((contract) => contract.status !== "ended");
  return <section className="subpanel">
    <h3>Umowa tygodniowa <PolicyHint>{`${formatMoney(policy.regular_lesson_price_minor, policy.currency)} za ${policy.lesson_duration_minutes} minut. Umowa kończy się ${policy.contract_end_day}.${policy.contract_end_month}. Limit przełożeń w miesiącu: ${policy.contract_monthly_reschedules}. Bezpłatne odwołania: ${policy.contract_free_cancellations}. Termin zastępczy: ${policy.contract_replacement_deadline_days} dni.`}</PolicyHint></h3>
    {active ? <ContractActions accountId={accountId} assignmentId={assignmentId} contract={active} policy={policy} /> : <ActivationFlow assignmentId={assignmentId} adHocLessons={adHocLessons} policy={policy} />}
  </section>;
}

function ActivationFlow({ assignmentId, adHocLessons, policy }: { assignmentId: string; adHocLessons: Lesson[]; policy: Policy }) {
  const mutation = usePlanMutation();
  const { notify } = useToast();
  const [startOn, setStartOn] = useState("");
  const [weekday, setWeekday] = useState("1");
  const [startTime, setStartTime] = useState("17:00");
  const [convert, setConvert] = useState<string[]>([]);
  const [outcomes, setOutcomes] = useState<Record<string, string>>({});
  const pastDates = pastOccurrences(startOn, Number(weekday));
  const summary = activationSummary(startOn, Number(weekday), startTime, policy, convert.length, pastDates.length);
  async function confirm() {
    await mutation.mutateAsync({ assignmentId, write: () => activateContract(assignmentId, {
      start_on: startOn, weekday: Number(weekday), start_time: startTime,
      ...(convert.length ? { convert_lesson_ids: convert } : {}),
      ...(pastDates.length ? { backdate_reason: "Umowa aktywowana wstecznie z potwierdzonymi wynikami.", past_outcomes: outcomeMap(pastDates, outcomes) } : {}),
    }) });
    notify("Umowa aktywowana.");
  }
  return <>
    <ApiFeedback error={mutation.error} />
    <FlowPanel title="Aktywacja umowy" openLabel="Aktywuj umowę" confirmLabel="Aktywuj umowę" summary={summary} busy={mutation.isPending} onConfirm={confirm}>
      <label htmlFor={`start-${assignmentId}`}>Data początku</label>
      <input id={`start-${assignmentId}`} type="date" value={startOn} onChange={(event) => setStartOn(event.target.value)} required />
      <label htmlFor={`weekday-${assignmentId}`}>Dzień tygodnia</label>
      <select id={`weekday-${assignmentId}`} value={weekday} onChange={(event) => setWeekday(event.target.value)}>{weekdayLabels.map((label, index) => <option key={label} value={index}>{label}</option>)}</select>
      <label htmlFor={`time-${assignmentId}`}>Godzina</label>
      <input id={`time-${assignmentId}`} type="time" step={policy.start_grid_minutes * 60} value={startTime} onChange={(event) => setStartTime(event.target.value)} required />
      <LessonPicker lessons={adHocLessons} selected={convert} onChange={setConvert} />
      <PastOutcomes dates={pastDates} values={outcomes} onChange={setOutcomes} />
    </FlowPanel>
  </>;
}

function activationSummary(startOn: string, weekday: number, startTime: string, policy: Policy, converted: number, backdated: number): string | null {
  if (!startOn || !startTime) return null;
  const first = firstOccurrence(startOn, weekday);
  if (!first) return null;
  const parts = [
    `Pierwsza lekcja: ${formatScheduleDate(`${first}T12:00:00Z`)}.`,
    `Stały termin: ${weekdayLabels[weekday]} o ${startTime}, ${policy.lesson_duration_minutes} minut.`,
    `Cena: ${formatMoney(policy.regular_lesson_price_minor, policy.currency)} za lekcję.`,
    `Umowa kończy się ${policy.contract_end_day}.${policy.contract_end_month}.`,
  ];
  if (converted) parts.push(`Zamieniamy ${converted} pojedynczych lekcji na lekcje z umowy.`);
  if (backdated) parts.push(`Aktywacja wsteczna obejmuje ${backdated} minionych terminów.`);
  return parts.join(" ");
}

// The contract runs weekly from the first matching weekday on or after the start date.
export function firstOccurrence(startOn: string, weekday: number): string | null {
  const start = Date.parse(`${startOn}T00:00:00Z`);
  if (Number.isNaN(start)) return null;
  const shift = (weekday - new Date(start).getUTCDay() + 7) % 7;
  return new Date(start + shift * 86_400_000).toISOString().slice(0, 10);
}

export function pastOccurrences(startOn: string, weekday: number, now = new Date()): string[] {
  const first = firstOccurrence(startOn, weekday);
  if (!first) return [];
  const dates: string[] = [];
  for (let value = Date.parse(`${first}T00:00:00Z`); value < now.getTime() && dates.length < 26; value += 7 * 86_400_000) {
    dates.push(new Date(value).toISOString().slice(0, 10));
  }
  return dates;
}

function outcomeMap(dates: string[], values: Record<string, string>): Record<string, string> {
  return Object.fromEntries(dates.map((date) => [date, values[date] ?? "completed"]));
}
