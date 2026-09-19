// Changes the recurring weekday and time from a chosen date, leaving earlier lessons untouched.

import { useState } from "react";

import { changeContractSchedule } from "../../../../api/commercial";
import type { Policy, RegularContract } from "../../../../api/contracts";
import { ApiFeedback } from "../../../../components/ApiFeedback";
import { FlowPanel } from "../../../../components/FlowPanel";
import { useToast } from "../../../../components/Toast";
import { usePlanMutation } from "../../../../query/commercial";
import { formatScheduleDate, weekdayLabels } from "../../../../time/schedule";

export function ScheduleChangeFlow({ assignmentId, contract, policy }: { assignmentId: string; contract: RegularContract; policy: Policy }) {
  const mutation = usePlanMutation();
  const { notify } = useToast();
  const [effectiveOn, setEffectiveOn] = useState("");
  const [weekday, setWeekday] = useState(String(contract.weekday));
  const [time, setTime] = useState(contract.start_time);
  const summary = effectiveOn ? `Od ${formatScheduleDate(`${effectiveOn}T12:00:00Z`)} stały termin to ${weekdayLabels[Number(weekday)]} o ${time}. Wcześniejsze lekcje zostają bez zmian.` : null;
  async function confirm() {
    await mutation.mutateAsync({ assignmentId, write: () => changeContractSchedule(contract.id, { effective_on: effectiveOn, weekday: Number(weekday), start_time: time }) });
    notify("Zmieniono stały termin.");
  }
  return <>
    <ApiFeedback error={mutation.error} />
    <FlowPanel title="Zmiana stałego terminu" openLabel="Zmień stały termin" confirmLabel="Zmień termin" summary={summary} busy={mutation.isPending} onConfirm={confirm}>
      <label htmlFor={`effective-${contract.id}`}>Obowiązuje od</label>
      <input id={`effective-${contract.id}`} type="date" value={effectiveOn} onChange={(event) => setEffectiveOn(event.target.value)} required />
      <label htmlFor={`weekday-${contract.id}`}>Dzień tygodnia</label>
      <select id={`weekday-${contract.id}`} value={weekday} onChange={(event) => setWeekday(event.target.value)}>{weekdayLabels.map((label, index) => <option key={label} value={index}>{label}</option>)}</select>
      <label htmlFor={`start-time-${contract.id}`}>Godzina</label>
      <input id={`start-time-${contract.id}`} type="time" step={policy.start_grid_minutes * 60} value={time} onChange={(event) => setTime(event.target.value)} required />
    </FlowPanel>
  </>;
}
