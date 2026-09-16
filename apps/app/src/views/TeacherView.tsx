// Composes the teacher calendar, commercial workspaces, financial queue, and availability editor.

import { useEffect, type ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { assignmentDisplayName } from "../api/scheduling";
import type { CalendarResponse, Policy } from "../api/contracts";
import { authenticatedRecord, personaDisplayName, usePersonaLogout, usePersonaSession } from "../auth/session";
import { ApiFeedback, LessonList } from "../components/ScheduleBits";
import { businessPolicyQuery } from "../query/commercial";
import { teacherCalendarQuery } from "../query/scheduling";
import { router } from "../router";
import { AvailabilityWorkspace } from "./teacher/AvailabilityWorkspace";
import { TeacherCommercialWorkspace } from "./teacher/TeacherCommercialWorkspace";
import { TeacherFinancialWorkspace } from "./teacher/TeacherFinancialWorkspace";

type TeacherViewProps = { availability?: boolean };

export function TeacherView({ availability = false }: TeacherViewProps) {
  const session = usePersonaSession("teacher");
  const record = authenticatedRecord(session.data);
  const calendar = useQuery(teacherCalendarQuery(record?.id));
  const policy = useQuery({ ...businessPolicyQuery("teacher"), enabled: Boolean(record) });
  const logout = usePersonaLogout("teacher");
  const unauthenticated = session.data?.kind === "unauthenticated";
  useEffect(() => { if (unauthenticated) void router.navigate({ to: "/teachers/login" }); }, [unauthenticated]);
  if (session.isError) return <main className="center-shell"><section className="status-card" role="alert"><h1>Nie możemy teraz sprawdzić sesji.</h1><button className="secondary-button" onClick={() => void session.refetch()}>Spróbuj ponownie</button></section></main>;
  if (!record) return null;
  async function handleLogout() { await logout.mutateAsync(); await router.navigate({ to: "/teachers/login" }); }
  const frame = (children: ReactNode) => <PanelFrame title={`Cześć, ${personaDisplayName(record)}.`} availability={availability} onLogout={() => void handleLogout()}>{children}</PanelFrame>;
  const panelError = [calendar.error, policy.error].find(Boolean);
  if (panelError) return frame(<ApiFeedback error={panelError} onRetry={() => { void calendar.refetch(); void policy.refetch(); }} />);
  if (!calendar.data || !policy.data) return frame(<p className="loading-state" role="status">Ładowanie kalendarza…</p>);
  if (availability) return frame(<AvailabilityWorkspace data={calendar.data} policy={policy.data} timezone={record.timezone || "Europe/Warsaw"} />);
  return frame(<TeacherDashboard accountId={record.id} data={calendar.data} policy={policy.data} />);
}

function TeacherDashboard({ accountId, data, policy }: { accountId: string; data: CalendarResponse; policy: Policy }) {
  const learnerNames = new Map(data.assignments.map((assignment) => [assignment.learner, assignmentDisplayName(assignment, "teacher")]));
  return <>
    <section className="panel-section"><p className="eyebrow">nutka / nauczyciel</p><h2>Kalendarz i rozliczenia</h2><p className="supporting-copy">Rezerwacje w {policy.booking_horizon_days} dniach są oddzielone od dalszych stałych terminów. Płatności miesięczne mają termin do {policy.monthly_payment_due_day}. dnia miesiąca.</p></section>
    <TeacherFinancialWorkspace accountId={accountId} />
    <TeacherCommercialWorkspace accountId={accountId} assignments={data.assignments} policy={policy} />
    <section className="panel-section"><h2>Lekcje w najbliższych {policy.booking_horizon_days} dniach</h2><LessonList lessons={data.near_term_lessons} role="teacher" policy={policy} commercialSummaries={data.commercial_summaries} counterpartNames={learnerNames} /></section>
    <section className="panel-section"><h2>Dalsze rezerwacje umowne</h2><LessonList lessons={data.later_contract_lessons ?? []} role="teacher" policy={policy} commercialSummaries={data.commercial_summaries} counterpartNames={learnerNames} /></section>
  </>;
}

function PanelFrame({ title, availability, onLogout, children }: { title: string; availability: boolean; onLogout: () => void; children: ReactNode }) {
  return <main className="panel-shell"><header className="panel-header"><div><p className="wordmark">nutka</p><p className="eyebrow">nutka / nauczyciel</p></div><nav className="panel-nav"><Link to="/teachers" className={!availability ? "active" : ""}>Kalendarz</Link><Link to="/teachers/availability" className={availability ? "active" : ""}>Dostępność</Link></nav><button className="text-button" onClick={onLogout}>Wyloguj</button></header><section className="panel-hero"><h1>{title}</h1><p>Zarządzaj planami, lekcjami i rozliczeniami uczniów.</p></section>{children}</main>;
}
