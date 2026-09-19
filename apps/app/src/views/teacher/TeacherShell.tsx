// Guards every teacher route with the teacher session and supplies the shared frame, navigation, calendar, and policy.

import { useEffect, type ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import type { CalendarResponse, Policy } from "../../api/contracts";
import { authenticatedRecord, personaDisplayName, usePersonaLogout, usePersonaSession } from "../../auth/session";
import { ApiFeedback } from "../../components/ApiFeedback";
import { PanelFrame } from "../../components/PanelFrame";
import { Skeleton } from "../../components/Skeleton";
import { businessPolicyQuery } from "../../query/commercial";
import { teacherCalendarQuery } from "../../query/scheduling";
import { router } from "../../router";

export type TeacherContext = { accountId: string; timezone: string; calendar: CalendarResponse; policy: Policy };

const screens = [
  { to: "/teachers", label: "Dziś" },
  { to: "/teachers/calendar", label: "Kalendarz" },
  { to: "/teachers/students", label: "Uczniowie" },
  { to: "/teachers/billing", label: "Rozliczenia" },
  { to: "/teachers/availability", label: "Dostępność" },
] as const;

export function TeacherShell({ lede, children }: { lede: string; children: (context: TeacherContext) => ReactNode }) {
  const session = usePersonaSession("teacher");
  const record = authenticatedRecord(session.data);
  const calendar = useQuery(teacherCalendarQuery(record?.id));
  const policy = useQuery({ ...businessPolicyQuery("teacher"), enabled: Boolean(record) });
  const logout = usePersonaLogout("teacher");
  const unauthenticated = session.data?.kind === "unauthenticated";
  useEffect(() => { if (unauthenticated) void router.navigate({ to: "/teachers/login" }); }, [unauthenticated]);
  if (session.isError) return <SessionUnavailable onRetry={() => void session.refetch()} />;
  if (!record) return null;
  async function handleLogout() { await logout.mutateAsync(); await router.navigate({ to: "/teachers/login" }); }
  const frame = (body: ReactNode) => <PanelFrame eyebrow="Panel nauczyciela" name={personaDisplayName(record)} lede={lede} nav={<TeacherNav />} onLogout={() => void handleLogout()}>{body}</PanelFrame>;
  const failure = [calendar.error, policy.error].find(Boolean);
  if (failure) return frame(<ApiFeedback error={failure} onRetry={() => { void calendar.refetch(); void policy.refetch(); }} />);
  if (!calendar.data || !policy.data) return frame(<Skeleton lines={5} label="Ładowanie panelu…" />);
  return frame(children({ accountId: record.id, timezone: record.timezone || "Europe/Warsaw", calendar: calendar.data, policy: policy.data }));
}

function SessionUnavailable({ onRetry }: { onRetry: () => void }) {
  return <main className="center-shell"><section className="status-card" role="alert"><h1>Nie możemy teraz sprawdzić sesji.</h1><button className="btn btn-ghost btn-sm" onClick={onRetry}>Spróbuj ponownie</button></section></main>;
}

function TeacherNav() {
  return <nav className="panel-nav" aria-label="Panel nauczyciela">{screens.map((screen) => <Link key={screen.to} to={screen.to} activeOptions={{ exact: screen.to === "/teachers" }}>{screen.label}</Link>)}</nav>;
}
