// Lists every learner with their plan state, so the roster costs one calendar read instead of four reads per learner.

import { Link } from "@tanstack/react-router";

import { planCopy } from "../../api/copy";
import { EmptyState } from "../../components/EmptyState";
import { rosterRows, type RosterRow } from "./roster";
import { TeacherShell } from "./TeacherShell";

const settlementCopy: Record<RosterRow["settlement"], string> = { settled: "Rozliczony", pending: "Oczekuje płatności", overdue: "Zaległość" };

export function StudentsView() {
  return <TeacherShell lede="Przejrzyj stan każdego ucznia i otwórz go, aby zmienić plan.">
    {(context) => <Roster rows={rosterRows(context.calendar, context.policy)} />}
  </TeacherShell>;
}

function Roster({ rows }: { rows: RosterRow[] }) {
  if (rows.length === 0) return <section className="panel-section"><h2>Uczniowie</h2><EmptyState action={<Link className="btn btn-ghost btn-sm" to="/teachers/availability">Ustaw dostępność</Link>}>Nie masz jeszcze uczniów.</EmptyState></section>;
  return <section className="panel-section"><h2>Uczniowie</h2><div className="assignment-list">{rows.map((row) => <RosterCard key={row.assignmentId} row={row} />)}</div></section>;
}

function RosterCard({ row }: { row: RosterRow }) {
  return <Link className="assignment-row inline-link" to="/teachers/students/$assignmentId" params={{ assignmentId: row.assignmentId }}>
    <div>
      <p className="eyebrow">Uczeń</p>
      <h3>{row.learnerName}</h3>
      <p className="lesson-meta">{row.plan ? planCopy[row.plan] : "Brak aktywnego planu"}{row.weeklySlot ? ` · ${row.weeklySlot}` : ""}{row.active ? "" : " · nieaktywny"}</p>
    </div>
    <span className={`status-badge status-${row.settlement}`}>{settlementCopy[row.settlement]}</span>
  </Link>;
}
