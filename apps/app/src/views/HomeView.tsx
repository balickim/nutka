// Renders the learner's horizon calendar, commercial balances, plan-aware booking, notice, teacher materials, and redacted history.

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { contractStatusCopy, planCopy, polishEvent, polishTokenState } from "../api/copy";
import { bookFlexibleLesson, submitContractNotice } from "../api/commercial";
import type { CommercialSummary, HistoryEvent, Policy } from "../api/contracts";
import { assignmentDisplayName, type Assignment, type CalendarResponse, type Slot } from "../api/scheduling";
import { authCopy } from "../auth/copy";
import { authenticatedRecord, personaDisplayName, usePersonaLogout, usePersonaSession } from "../auth/session";
import { ApiFeedback } from "../components/ApiFeedback";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { EmptyState } from "../components/EmptyState";
import { Skeleton } from "../components/Skeleton";
import { useToast } from "../components/Toast";
import { formatMoney } from "../money";
import { LessonList } from "../components/ScheduleBits";
import { MaterialList } from "../components/MaterialList";
import { businessPolicyQuery, commercialSummaryQuery, historyQuery, useBookingMutation, usePlanMutation } from "../query/commercial";
import { materialsQuery } from "../query/materials";
import { learnerCalendarQuery, useLearnerSlots } from "../query/scheduling";
import { router } from "../router";
import { formatScheduleDate, formatScheduleInstant, groupSlotsByLocalDate } from "../time/schedule";

export function HomeView() {
  const session = usePersonaSession("learner");
  const record = authenticatedRecord(session.data);
  const calendar = useQuery(learnerCalendarQuery(record?.id));
  const policy = useQuery({ ...businessPolicyQuery("learner"), enabled: Boolean(record) });
  const assignments = activeOf(calendar.data);
  const slots = useLearnerSlots(record?.id, assignments);
  const logout = usePersonaLogout("learner");
  const unauthenticated = session.data?.kind === "unauthenticated";
  useEffect(() => { if (unauthenticated) void router.navigate({ to: "/learners/login" }); }, [unauthenticated]);
  if (session.isError) return <SessionUnavailable onRetry={() => void session.refetch()} />;
  if (!record) return null;
  async function handleLogout() { await logout.mutateAsync(); await router.navigate({ to: "/learners/login" }); }
  const frame = (children: React.ReactNode) => <PanelFrame title={`Cześć, ${personaDisplayName(record)}.`} onLogout={() => void handleLogout()}>{children}</PanelFrame>;
  const panelError = [calendar.error, policy.error, slots.error].find(Boolean);
  if (panelError) return frame(<ApiFeedback error={panelError} onRetry={() => { void calendar.refetch(); void policy.refetch(); slots.refetch(); }} />);
  if (!calendar.data || !policy.data || slots.pending) return frame(<Skeleton lines={5} label="Ładowanie kalendarza…" />);
  return frame(<LearnerDashboard accountId={record.id} calendar={calendar.data} assignments={assignments} slots={slots} policy={policy.data} />);
}

function LearnerDashboard({ accountId, calendar, assignments, slots, policy }: { accountId: string; calendar: CalendarResponse; assignments: Assignment[]; slots: ReturnType<typeof useLearnerSlots>; policy: Policy }) {
  const names = new Map(calendar.assignments.map((assignment) => [assignment.teacher, assignmentDisplayName(assignment, "learner")]));
  return <>
    <section className="panel-section"><p className="eyebrow">nutka / uczeń</p><h2>Twój plan i rezerwacje</h2><p className="supporting-copy">Terminy obejmują starty w najbliższych {policy.booking_horizon_days} dniach. Lekcja trwa {policy.lesson_duration_minutes} minut, a kalendarz chroni dodatkowo {policy.participant_buffer_minutes} minut przed i po niej.</p></section>
    <section className="panel-section"><div className="assignment-grid"><AssignmentCards accountId={accountId} assignments={assignments} slots={slots} policy={policy} /></div></section>
    <section className="panel-section"><h2>Lekcje w horyzoncie</h2><LessonList lessons={calendar.near_term_lessons} role="learner" policy={policy} commercialSummaries={calendar.commercial_summaries} counterpartNames={names} /></section>
  </>;
}

function AssignmentCards({ accountId, assignments, slots, policy }: { accountId: string; assignments: Assignment[]; slots: ReturnType<typeof useLearnerSlots>; policy: Policy }) {
  if (assignments.length === 0) return <EmptyState>Nie masz aktywnych przypisań.</EmptyState>;
  return assignments.map((assignment) => <LearnerAssignmentCard key={assignment.id} accountId={accountId} assignment={assignment} slots={slots.slotsFor(assignment.id)} policy={policy} />);
}

function LearnerAssignmentCard({ accountId, assignment, slots, policy }: { accountId: string; assignment: Assignment; slots: Slot[]; policy: Policy }) {
  const summary = useQuery(commercialSummaryQuery("learner", accountId, assignment.id));
  const history = useQuery(historyQuery("learner", accountId, assignment.id));
  const booking = useBookingMutation();
  const plan = usePlanMutation();
  const { notify } = useToast();
  const [busy, setBusy] = useState<string | null>(null);
  const [noticeOpen, setNoticeOpen] = useState(false);
  const contract = summary.data?.contract;
  async function book(slot: Slot) {
    setBusy(slot.start_at);
    try {
      await booking.mutateAsync({ assignmentId: assignment.id, write: () => bookFlexibleLesson("learner", assignment.id, { start_at: slot.start_at }) });
      notify(`Zarezerwowano lekcję ${formatScheduleInstant(slot.start_at)}.`);
    } finally { setBusy(null); }
  }
  async function notice() {
    if (!contract) return;
    await plan.mutateAsync({ assignmentId: assignment.id, write: () => submitContractNotice("learner", contract.id) });
    notify("Wypowiedzenie złożone.");
    setNoticeOpen(false);
  }
  return <article className="assignment-card">
    <div className="assignment-heading"><div><p className="eyebrow">Nauczyciel</p><h3>{assignmentDisplayName(assignment, "learner")}</h3></div><span className="duration-badge">{policy.lesson_duration_minutes} min</span></div>
    <ApiFeedback error={summary.error || history.error || booking.error || plan.error} />
    <CommercialSummaryPanel details={summary.data} pending={summary.isPending} onNotice={() => setNoticeOpen(true)} />
    <ConfirmDialog open={noticeOpen} title="Wypowiedzenie umowy" consequence={contract ? `Umowa zakończy się ${formatScheduleDate(`${contract.end_on}T12:00:00Z`)}. Lekcje po tej dacie znikną z kalendarza.` : ""} confirmLabel="Złóż wypowiedzenie" danger busy={plan.isPending} onConfirm={() => void notice()} onCancel={() => setNoticeOpen(false)} />
    <BookingPanel regular={summary.data?.active_plan === "regular_contract"} slots={slots} busy={busy} policy={policy} onBook={(slot) => void book(slot)} />
    <LearnerMaterials accountId={accountId} assignmentId={assignment.id} />
    <LearnerHistory items={history.data?.items ?? []} />
    <TokenDetails details={summary.data} />
  </article>;
}

function CommercialSummaryPanel({ details, pending, onNotice }: { details?: CommercialSummary; pending: boolean; onNotice: () => void }) {
  if (pending || !details) return <Skeleton lines={3} label="Ładowanie planu…" />;
  return <div className="commercial-summary"><strong>{details.active_plan ? planCopy[details.active_plan] : "Brak aktywnego planu"}</strong>{details.package ? <p>Pakiet ważny do {details.package.valid_through}. Dostępne tokeny: {details.package.token_balance.available}, zarezerwowane: {details.package.token_balance.reserved}.</p> : null}{details.contract ? <p>Umowa: {details.contract.start_on}–{details.contract.end_on}. Stan: {contractStatusCopy[details.contract.status]}. Cena: {formatMoney(details.contract.price_minor, details.contract.currency)} za lekcję. Zmiany w miesiącu: {details.contract.remaining_monthly_reschedules}. Bezpłatne odwołania: {details.contract.remaining_free_cancellations}.</p> : null}<p>Oczekujące płatności: {details.payments.pending}. Nieopłacone: {details.payments.intentionally_unpaid}. Kredyt: {formatMoney(details.payments.credit_minor, details.payments.currency)}.</p>{details.contract ? <button className="text-button danger-button" onClick={onNotice}>Złóż wypowiedzenie</button> : null}</div>;
}

function BookingPanel({ regular, slots, busy, policy, onBook }: { regular: boolean; slots: Slot[]; busy: string | null; policy: Policy; onBook: (slot: Slot) => void }) {
  if (regular) return <p className="supporting-copy">Stałe terminy wynikają z umowy. Elastyczna rezerwacja jest wyłączona.</p>;
  return <><p className="supporting-copy">Rezerwacja wymaga {policy.learner_booking_minimum_hours} godz. wyprzedzenia, mieści się w horyzoncie {policy.booking_horizon_days} dni i zaczyna na siatce co {policy.start_grid_minutes} minut.</p><SlotPicker slots={slots} busy={busy} onBook={onBook} /></>;
}

function LearnerMaterials({ accountId, assignmentId }: { accountId: string; assignmentId: string }) {
  const materials = useQuery(materialsQuery("learner", accountId, assignmentId));
  return <section className="learner-materials"><h4>Materiały od nauczyciela</h4>
    {materials.error ? <ApiFeedback error={materials.error} onRetry={() => void materials.refetch()} /> : null}
    {materials.isPending ? <Skeleton lines={3} label="Ładowanie materiałów…" /> : <MaterialList items={materials.data?.items ?? []} empty="Nauczyciel nie dodał jeszcze materiałów." />}
  </section>;
}

function LearnerHistory({ items }: { items: HistoryEvent[] }) {
  return <details><summary>Historia</summary>{items.length ? <ul className="history-list">{items.map((event) => <li key={event.id}><strong>{polishEvent(event.event_type)}</strong> · {formatScheduleInstant(event.event_at)}{event.corrects_event ? " · korekta" : ""}</li>)}</ul> : <p className="supporting-copy">Brak zdarzeń.</p>}</details>;
}

function TokenDetails({ details }: { details?: CommercialSummary }) {
  if (!details?.package) return null;
  return <details><summary>Stany tokenów</summary><p className="supporting-copy">{Object.entries(details.package.token_balance).map(([state, count]) => `${polishTokenState(state)}: ${count}`).join(" · ")}</p></details>;
}

function SlotPicker({ slots, busy, onBook }: { slots: Slot[]; busy: string | null; onBook: (slot: Slot) => void }) {
  const groups = groupSlotsByLocalDate(slots);
  if (slots.length === 0) return <EmptyState>Brak wolnych terminów w horyzoncie.</EmptyState>;
  return <div className="slot-groups">{Array.from(groups.entries()).map(([date, items]) => <div className="slot-group" key={date}><h4>{formatScheduleDate(items[0].start_at)}</h4><div className="slot-grid">{items.map((slot) => <button className="slot-button" key={slot.start_at} disabled={busy === slot.start_at} onClick={() => onBook(slot)}>{busy === slot.start_at ? "Zapisywanie…" : formatScheduleInstant(slot.start_at)}<span>{slot.duration_minutes} min · plan dobierze system</span></button>)}</div></div>)}</div>;
}

function activeOf(calendar: CalendarResponse | undefined): Assignment[] {
  return calendar?.assignments.filter((assignment) => assignment.active) ?? [];
}

function SessionUnavailable({ onRetry }: { onRetry: () => void }) {
  return <main className="center-shell"><section className="status-card" role="alert"><h1>{authCopy.unavailable}</h1><button className="secondary-button" onClick={onRetry}>{authCopy.retry}</button></section></main>;
}

function PanelFrame({ title, onLogout, children }: { title: string; onLogout: () => void; children: React.ReactNode }) {
  return <main className="panel-shell"><header className="panel-header"><div className="panel-identity"><p className="wordmark">nutka</p><h1>{title}</h1></div><button className="text-button" onClick={onLogout}>{authCopy.logout}</button></header><p className="panel-lede">Planuj lekcje z przypisanymi nauczycielami.</p>{children}</main>;
}
