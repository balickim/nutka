// Books a lesson for the learner, offering only starts the policy allows and confirming a short-notice start.

import { useState } from "react";

import { bookFlexibleLesson } from "../../../api/commercial";
import type { Policy } from "../../../api/contracts";
import { ApiFeedback } from "../../../components/ApiFeedback";
import { FlowPanel } from "../../../components/FlowPanel";
import { PolicyHint } from "../../../components/PolicyHint";
import { useToast } from "../../../components/Toast";
import { useBookingMutation } from "../../../query/commercial";
import { formatScheduleInstant, localInputToUtc, utcToLocalInput } from "../../../time/schedule";

export function BookingPanel({ assignmentId, policy }: { assignmentId: string; policy: Policy }) {
  const mutation = useBookingMutation();
  const { notify } = useToast();
  const [start, setStart] = useState("");
  const [confirmShortNotice, setConfirmShortNotice] = useState(false);
  const shortNotice = start ? Date.parse(localInputToUtc(start)) - Date.now() < policy.learner_booking_minimum_hours * 3_600_000 : false;
  const blocked = shortNotice && !confirmShortNotice;
  const summary = start && !blocked ? `Lekcja ${formatScheduleInstant(localInputToUtc(start))}, ${policy.lesson_duration_minutes} minut. Sposób rozliczenia dobierzemy automatycznie.` : null;
  async function confirm() {
    await mutation.mutateAsync({ assignmentId, write: () => bookFlexibleLesson("teacher", assignmentId, { start_at: localInputToUtc(start), ...(confirmShortNotice ? { confirm_short_notice: true } : {}) }) });
    notify(`Zarezerwowano lekcję ${formatScheduleInstant(localInputToUtc(start))}.`);
    setStart("");
    setConfirmShortNotice(false);
  }
  return <section className="subpanel">
    <h3>Rezerwacja za ucznia <PolicyHint>{`Termin musi przypadać w najbliższych ${policy.booking_horizon_days} dniach. Lekcje zaczynają się co ${policy.start_grid_minutes} minut. Przed lekcją i po niej zostaje ${policy.participant_buffer_minutes} minut przerwy.`}</PolicyHint></h3>
    <ApiFeedback error={mutation.error} />
    <FlowPanel title="Rezerwacja lekcji" openLabel="Zarezerwuj termin" confirmLabel="Zarezerwuj" summary={summary} busy={mutation.isPending} onConfirm={confirm}>
      <label htmlFor={`booking-${assignmentId}`}>Termin</label>
      <input id={`booking-${assignmentId}`} type="datetime-local" step={policy.start_grid_minutes * 60} min={utcToLocalInput(new Date().toISOString())} max={utcToLocalInput(new Date(Date.now() + policy.booking_horizon_days * 86_400_000).toISOString())} value={start} onChange={(event) => { setStart(event.target.value); setConfirmShortNotice(false); }} />
      {shortNotice ? <label className="check-label"><input type="checkbox" checked={confirmShortNotice} onChange={(event) => setConfirmShortNotice(event.target.checked)} /> Potwierdzam termin krótszy niż {policy.learner_booking_minimum_hours} godz.</label> : null}
    </FlowPanel>
  </section>;
}
