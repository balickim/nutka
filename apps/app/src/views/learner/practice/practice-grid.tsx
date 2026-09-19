// Shows the last four weeks with a dot on each practice day. It has no score and no streak.

import { formatLocalDate } from "../../../time/schedule";
import { practiceGrid } from "./days";

const weekdays = ["Pn", "Wt", "Śr", "Cz", "Pt", "Sb", "Nd"];

export function PracticeGrid({ today, days }: { today: string; days: string[] }) {
  const weeks = practiceGrid(today, days);
  return <div className="practice-grid" role="table" aria-label="Dni z ćwiczeniem w ostatnich czterech tygodniach">
    <div className="practice-week" role="row">{weekdays.map((label) => <span key={label} role="columnheader">{label}</span>)}</div>
    {weeks.map((week) => <div className="practice-week" role="row" key={week[0].date}>
      {week.map((day) => <span key={day.date} role="cell" className={`practice-day${day.practiced ? " practiced" : ""}${day.future ? " future" : ""}${day.date === today ? " today" : ""}`} title={formatLocalDate(day.date)} aria-label={`${formatLocalDate(day.date)}${day.practiced ? ": ćwiczenie" : ""}`}>{Number(day.date.slice(8))}</span>)}
    </div>)}
  </div>;
}
