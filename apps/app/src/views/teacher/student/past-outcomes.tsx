// Records the outcome of each past date a backdated contract covers, one control per date.

import { outcomeCopy } from "../../../api/copy";

export type PastOutcome = { date: string; outcome: string };

const choices = [
  { value: "completed", label: outcomeCopy.completed },
  { value: "cancelled", label: "Odwołana" },
  { value: "learner_no_show", label: outcomeCopy.learner_no_show },
  { value: "scheduled", label: "Zaplanowana" },
];

export function PastOutcomes({ dates, values, onChange }: { dates: string[]; values: Record<string, string>; onChange: (value: Record<string, string>) => void }) {
  if (dates.length === 0) return null;
  return <fieldset className="picker">
    <legend>Wyniki lekcji przed dzisiejszą datą</legend>
    {dates.map((date) => <label key={date}>{date}
      <select value={values[date] ?? "completed"} onChange={(event) => onChange({ ...values, [date]: event.target.value })}>
        {choices.map((choice) => <option key={choice.value} value={choice.value}>{choice.label}</option>)}
      </select>
    </label>)}
  </fieldset>;
}
