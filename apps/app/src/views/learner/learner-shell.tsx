// Guards every learner route with the learner session and supplies the frame, navigation, calendar, policy, and selected assignment.

import { useEffect, type ReactNode } from "react";
import { Link, useLocation, useNavigate, useSearch } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import type { Assignment, CalendarResponse, Policy } from "../../api/contracts";
import { assignmentDisplayName } from "../../api/scheduling";
import { authCopy } from "../../auth/copy";
import { authenticatedRecord, personaDisplayName, usePersonaLogout, usePersonaSession } from "../../auth/session";
import { ApiFeedback } from "../../components/api-feedback";
import { EmptyState } from "../../components/empty-state";
import { PanelFrame } from "../../components/panel-frame";
import { Skeleton } from "../../components/skeleton";
import { businessPolicyQuery } from "../../query/commercial";
import { learnerCalendarQuery } from "../../query/scheduling";
import { router } from "../../router";
import { activeAssignments, selectAssignment } from "./assignment";

export type LearnerContext = { accountId: string; calendar: CalendarResponse; policy: Policy; assignment: Assignment };

const screens = [
  { to: "/learners", label: "Start" },
  { to: "/learners/lessons", label: "Lekcje" },
  { to: "/learners/pieces", label: "Utwory" },
] as const;

export function LearnerShell({ lede, children }: { lede: string; children: (context: LearnerContext) => ReactNode }) {
  const session = usePersonaSession("learner");
  const record = authenticatedRecord(session.data);
  const calendar = useQuery(learnerCalendarQuery(record?.id));
  const policy = useQuery({ ...businessPolicyQuery("learner"), enabled: Boolean(record) });
  const logout = usePersonaLogout("learner");
  const requested = useSearch({ strict: false }).a;
  const unauthenticated = session.data?.kind === "unauthenticated";
  useEffect(() => { if (unauthenticated) void router.navigate({ to: "/learners/login" }); }, [unauthenticated]);
  if (session.isError) return <SessionUnavailable onRetry={() => void session.refetch()} />;
  if (!record) return null;
  async function handleLogout() { await logout.mutateAsync(); await router.navigate({ to: "/learners/login" }); }
  const active = activeAssignments(calendar.data?.assignments ?? []);
  const assignment = selectAssignment(active, requested);
  const nav = <LearnerNav search={requested ? { a: requested } : {}} />;
  const frame = (body: ReactNode) => <div className="learner-panel"><PanelFrame eyebrow="Panel ucznia" name={personaDisplayName(record)} lede={lede} nav={nav} onLogout={() => void handleLogout()}>{active.length > 1 && assignment ? <AssignmentSwitcher active={active} selected={assignment} /> : null}{body}</PanelFrame></div>;
  const failure = [calendar.error, policy.error].find(Boolean);
  if (failure) return frame(<ApiFeedback error={failure} onRetry={() => { void calendar.refetch(); void policy.refetch(); }} />);
  if (!calendar.data || !policy.data) return frame(<Skeleton lines={5} label="Ładowanie panelu…" />);
  if (!assignment) return frame(<EmptyState>Nie masz jeszcze aktywnych zajęć. Skontaktuj się z nauczycielem.</EmptyState>);
  return frame(children({ accountId: record.id, calendar: calendar.data, policy: policy.data, assignment }));
}

function LearnerNav({ search }: { search: { a?: string } }) {
  return <nav className="panel-nav learner-nav" aria-label="Panel ucznia">{screens.map((screen) => <Link key={screen.to} to={screen.to} search={search} activeOptions={{ exact: screen.to === "/learners", includeSearch: false }}>{screen.label}</Link>)}</nav>;
}

function AssignmentSwitcher({ active, selected }: { active: Assignment[]; selected: Assignment }) {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  return <div className="assignment-switcher"><label htmlFor="learner-assignment">Nauczyciel</label>
    <select id="learner-assignment" value={selected.id} onChange={(event) => void navigate({ to: pathname, search: { a: event.target.value } })}>{active.map((assignment) => <option key={assignment.id} value={assignment.id}>{assignmentDisplayName(assignment, "learner")}</option>)}</select>
  </div>;
}

function SessionUnavailable({ onRetry }: { onRetry: () => void }) {
  return <main className="center-shell"><section className="status-card" role="alert"><h1>{authCopy.unavailable}</h1><button className="btn btn-ghost btn-sm" onClick={onRetry}>{authCopy.retry}</button></section></main>;
}
