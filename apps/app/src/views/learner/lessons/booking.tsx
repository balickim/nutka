// Offers flexible booking for one assignment: the learner picks a slot and confirms it in a dialog before the write.

import { useState } from "react";

import { bookFlexibleLesson } from "../../../api/commercial";
import type { Assignment, Policy, Slot } from "../../../api/contracts";
import { assignmentDisplayName } from "../../../api/scheduling";
import { ApiFeedback } from "../../../components/api-feedback";
import { ConfirmDialog } from "../../../components/confirm-dialog";
import { EmptyState } from "../../../components/empty-state";
import { useToast } from "../../../components/toast";
import { useBookingMutation } from "../../../query/commercial";
import { formatScheduleDate, formatScheduleInstant, formatScheduleTime, groupSlotsByLocalDate } from "../../../time/schedule";

export function LearnerBooking({ assignment, slots, policy }: { assignment: Assignment; slots: Slot[]; policy: Policy }) {
  const booking = useBookingMutation();
  const { notify } = useToast();
  const [picked, setPicked] = useState<Slot | null>(null);
  async function book(slot: Slot) {
    try {
      await booking.mutateAsync({ assignmentId: assignment.id, write: () => bookFlexibleLesson("learner", assignment.id, { start_at: slot.start_at }) });
      notify(`Zarezerwowano lekcję ${formatScheduleInstant(slot.start_at)}.`);
    } finally { setPicked(null); }
  }
  const consequence = picked ? `${formatScheduleInstant(picked.start_at)}, ${picked.duration_minutes} minut, nauczyciel: ${assignmentDisplayName(assignment, "learner")}. Sposób rozliczenia dobierzemy automatycznie.` : "";
  return <details className="booking-details"><summary>Zarezerwuj lekcję</summary>
    <p className="supporting-copy">Rezerwacja wymaga {policy.learner_booking_minimum_hours} godz. wyprzedzenia i mieści się w najbliższych {policy.booking_horizon_days} dniach. Wybierz termin, a potwierdzisz go w następnym kroku.</p>
    <ApiFeedback error={booking.error} />
    <SlotPicker slots={slots} onPick={setPicked} />
    <ConfirmDialog open={picked !== null} title="Rezerwacja lekcji" consequence={consequence} confirmLabel="Zarezerwuj lekcję" busy={booking.isPending} onConfirm={() => { if (picked) void book(picked).catch(() => undefined); }} onCancel={() => setPicked(null)} />
  </details>;
}

function SlotPicker({ slots, onPick }: { slots: Slot[]; onPick: (slot: Slot) => void }) {
  const groups = groupSlotsByLocalDate(slots);
  if (slots.length === 0) return <EmptyState>Brak wolnych terminów w najbliższych dniach.</EmptyState>;
  return <div className="slot-groups">{Array.from(groups.entries()).map(([date, items]) => <div className="slot-group" key={date}><h4>{formatScheduleDate(items[0].start_at)}</h4><div className="slot-grid">{items.map((slot) => <button className="slot-button" key={slot.start_at} onClick={() => onPick(slot)}>{formatScheduleTime(slot.start_at)}<span>{slot.duration_minutes} min</span></button>)}</div></div>)}</div>;
}
