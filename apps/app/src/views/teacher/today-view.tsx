// Shows only dated work: the lessons of the current local day and the items that wait for a decision.

import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import type { CalendarResponse, Lesson, Policy, UnresolvedWorkResponse } from "../../api/contracts";
import { EmptyState } from "../../components/empty-state";
import { Skeleton } from "../../components/skeleton";
import { ApiFeedback } from "../../components/api-feedback";
import { formatMoney } from "../../money";
import { unresolvedWorkQuery } from "../../query/commercial";
import { daySummariesQuery } from "../../query/practice";
import { formatScheduleInstant, isSameLocalDay } from "../../time/schedule";
import { TodayLessonCard } from "./today/today-lesson-card";
import { uniqueCharges } from "./billing-view";
import { learnerNames } from "./roster";
import { TeacherShell } from "./teacher-shell";

export function TodayView() {
  return <TeacherShell lede="Twoje dzisiejsze lekcje i sprawy, które czekają na decyzję.">
    {(context) => <Today accountId={context.accountId} calendar={context.calendar} policy={context.policy} />}
  </TeacherShell>;
}

function Today({ accountId, calendar, policy }: { accountId: string; calendar: CalendarResponse; policy: Policy }) {
  const names = learnerNames(calendar);
  return <>
    <TodayLessons accountId={accountId} lessons={todayLessons(calendar)} names={names} policy={policy} />
    <DecisionQueue accountId={accountId} names={names} />
  </>;
}

export function todayLessons(calendar: CalendarResponse, now = new Date()): Lesson[] {
  return calendar.near_term_lessons
    .filter((lesson) => lesson.schedule_state === "scheduled" && isSameLocalDay(lesson.start_at, now))
    .sort((left, right) => left.start_at.localeCompare(right.start_at));
}

// One request returns the practice summaries of every lesson today. A failed read only hides the summaries.
function TodayLessons({ accountId, lessons, names, policy }: { accountId: string; lessons: Lesson[]; names: ReadonlyMap<string, string>; policy: Policy }) {
  const practice = useQuery({ ...daySummariesQuery(accountId, localDate()), enabled: lessons.length > 0 });
  const summaries = new Map(practice.data?.items.map((item) => [item.lesson, item.summary]));
  return <section className="panel-section">
    <h2>Dzisiejsze lekcje</h2>
    {lessons.length === 0
      ? <EmptyState action={<Link className="btn btn-ghost btn-sm" to="/teachers/calendar">Otwórz kalendarz</Link>}>Dziś nie masz lekcji.</EmptyState>
      : <div className="lesson-list">{lessons.map((lesson) => <TodayLessonCard key={lesson.id} accountId={accountId} lesson={lesson} learner={names.get(lesson.assignment)} policy={policy} practice={summaries.get(lesson.id)} />)}</div>}
  </section>;
}

function localDate(now = new Date()): string {
  return new Intl.DateTimeFormat("en-CA", { year: "numeric", month: "2-digit", day: "2-digit" }).format(now);
}

function DecisionQueue({ accountId, names }: { accountId: string; names: ReadonlyMap<string, string> }) {
  const unresolved = useQuery(unresolvedWorkQuery(accountId));
  return <section className="panel-section">
    <div className="section-heading"><h2>Wymaga decyzji</h2><Link className="text-button" to="/teachers/billing">Otwórz rozliczenia</Link></div>
    <ApiFeedback error={unresolved.error} onRetry={() => void unresolved.refetch()} />
    {unresolved.error ? null : <QueueList data={unresolved.data} pending={unresolved.isPending} names={names} />}
  </section>;
}

type DecisionItem = { id: string; text: string; actionable: boolean };

export function decisionItems(data: UnresolvedWorkResponse, names: ReadonlyMap<string, string>): DecisionItem[] {
  const learner = (assignment: string) => names.get(assignment) ?? "Uczeń";
  return [
    ...data.awaiting_outcome.map((item) => ({ id: item.lesson, text: `${learner(item.assignment)} · lekcja bez wyniku, zakończona ${formatScheduleInstant(item.ended_at)}`, actionable: false })),
    ...data.pending_settlements.map((item) => ({ id: item.lesson, text: `${learner(item.assignment)} · lekcja do rozliczenia, ${formatMoney(item.amount_minor, item.currency)}`, actionable: true })),
    ...uniqueCharges(data).map((charge) => ({ id: charge.id, text: `${learner(charge.assignment)} · ${charge.overdue ? "zaległa należność" : "należność"}, ${formatMoney(charge.current_amount_minor, charge.currency)}`, actionable: true })),
  ];
}

function QueueList({ data, pending, names }: { data?: UnresolvedWorkResponse; pending: boolean; names: ReadonlyMap<string, string> }) {
  if (pending || !data) return <Skeleton lines={3} label="Ładowanie kolejki…" />;
  const items = decisionItems(data, names);
  if (items.length === 0) return <EmptyState action={<Link className="btn btn-ghost btn-sm" to="/teachers/students">Otwórz listę uczniów</Link>}>Nic nie czeka na Twoją decyzję.</EmptyState>;
  return <ul className="decision-queue">{items.map((item) => <li key={item.id}><span>{item.text}</span>{item.actionable ? <Link className="btn btn-ghost btn-sm" to="/teachers/billing">Rozlicz</Link> : null}</li>)}</ul>;
}
