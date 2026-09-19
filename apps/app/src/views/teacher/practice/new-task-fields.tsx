// Holds the fields of one new practice task: title, details, suggested minutes, and an optional piece and material of the same learner.

import { useQuery } from "@tanstack/react-query";

import { practiceLimits, type NewTask } from "../../../api/practice";
import { materialsQuery } from "../../../query/materials";
import { piecesQuery } from "../../../query/pieces";

export const emptyTask = (): NewTask => ({ title: "", details: "", suggested_minutes: null, piece: "", material: "" });

type FieldsProps = { index: number; accountId: string; assignmentId: string; value: NewTask; onChange: (value: NewTask) => void };

export function NewTaskFields({ index, accountId, assignmentId, value, onChange }: FieldsProps) {
  const id = (field: string) => `task-${index}-${field}`;
  const set = (patch: Partial<NewTask>) => onChange({ ...value, ...patch });
  return <fieldset className="picker new-task"><legend>Nowe zadanie {index + 1}</legend>
    <label htmlFor={id("title")}>Co ćwiczyć</label>
    <input id={id("title")} value={value.title} maxLength={practiceLimits.titleMax} placeholder="Na przykład: refren, tempo 70" onChange={(event) => set({ title: event.target.value })} />
    <label htmlFor={id("details")}>Szczegóły (opcjonalnie)</label>
    <textarea id={id("details")} rows={2} maxLength={practiceLimits.detailsMax} value={value.details} onChange={(event) => set({ details: event.target.value })} />
    <label htmlFor={id("minutes")}>Minut dziennie (opcjonalnie)</label>
    <input id={id("minutes")} type="number" inputMode="numeric" min={1} max={practiceLimits.suggestedMinutesMax} value={value.suggested_minutes ?? ""} onChange={(event) => set({ suggested_minutes: event.target.value ? Number(event.target.value) : null })} />
    <LinkSelects accountId={accountId} assignmentId={assignmentId} value={value} id={id} set={set} />
  </fieldset>;
}

type LinkProps = { accountId: string; assignmentId: string; value: NewTask; id: (field: string) => string; set: (patch: Partial<NewTask>) => void };

function LinkSelects({ accountId, assignmentId, value, id, set }: LinkProps) {
  const pieces = useQuery(piecesQuery("teacher", accountId, assignmentId)).data?.items ?? [];
  const materials = useQuery(materialsQuery("teacher", accountId, assignmentId)).data?.items ?? [];
  return <>
    <LinkSelect id={id("piece")} label="Utwór" none="Bez utworu" value={value.piece} options={pieces} onChange={(piece) => set({ piece })} />
    <LinkSelect id={id("material")} label="Materiał" none="Bez materiału" value={value.material} options={materials} onChange={(material) => set({ material })} />
  </>;
}

function LinkSelect({ id, label, none, value, options, onChange }: { id: string; label: string; none: string; value: string; options: { id: string; title: string }[]; onChange: (value: string) => void }) {
  if (options.length === 0) return null;
  return <><label htmlFor={id}>{label}</label><select id={id} value={value} onChange={(event) => onChange(event.target.value)}><option value="">{none}</option>{options.map((option) => <option key={option.id} value={option.id}>{option.title}</option>)}</select></>;
}
