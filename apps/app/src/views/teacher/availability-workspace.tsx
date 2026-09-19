// Edits teacher availability only through an impact review dialog, explicit conflict resolutions, and one all-or-nothing save.

import { useState, type FormEvent } from "react";

import { commitAvailability, previewAvailability } from "../../api/commercial";
import type { AvailabilityConflictResolution, AvailabilityPreview, AvailabilityProposal, CalendarResponse, Policy } from "../../api/contracts";
import { availabilityHelpCopy } from "../../api/copy";
import { ApiFeedback } from "../../components/api-feedback";
import { ConfirmDialog } from "../../components/confirm-dialog";
import { EmptyState } from "../../components/empty-state";
import { HelpHeading } from "../../components/help-heading";
import { LessonList } from "../../components/schedule-bits";
import { Tooltip } from "../../components/tooltip";
import { useAvailabilityCommitMutation } from "../../query/commercial";
import { formatScheduleInstant, futureExceptionDraft, localInputToUtc, weekdayLabels } from "../../time/schedule";

export function AvailabilityWorkspace({ data, policy, timezone }: { data: CalendarResponse; policy: Policy; timezone: string }) {
  const [preview, setPreview] = useState<AvailabilityPreview | null>(null);
  const [proposal, setProposal] = useState<AvailabilityProposal | null>(null);
  const [resolutions, setResolutions] = useState<Record<string, AvailabilityConflictResolution>>({});
  const [previewError, setPreviewError] = useState<unknown>(null);
  const commit = useAvailabilityCommitMutation();
  async function inspect(next: AvailabilityProposal) {
    setPreviewError(null);
    try {
      const result = await previewAvailability(next);
      setProposal(next);
      setPreview(result);
      setResolutions({});
    } catch (error) { setPreviewError(error); }
  }
  async function save() {
    if (!preview || !proposal) return;
    await commit.mutateAsync({ write: () => commitAvailability({ proposal, preview_version: preview.preview_version, resolutions: Object.values(resolutions) }) });
    setPreview(null);
    setProposal(null);
  }
  return <>
    <RulePanel data={data} policy={policy} timezone={timezone} onPreview={inspect} error={previewError || commit.error} />
    <ExceptionPanel data={data} policy={policy} onPreview={inspect} />
    {preview && proposal ? <ReviewDialog preview={preview} horizonDays={policy.booking_horizon_days} resolutions={resolutions} onResolution={(value) => setResolutions((current) => ({ ...current, [value.lesson]: value }))} onSave={() => void save()} onClose={() => setPreview(null)} busy={commit.isPending} /> : null}
    <AvailabilityLessons data={data} policy={policy} />
  </>;
}
function RulePanel({ data, policy, timezone, onPreview, error }: { data: CalendarResponse; policy: Policy; timezone: string; onPreview: (value: AvailabilityProposal) => Promise<void>; error: unknown }) {
  const help = availabilityHelpCopy(policy);
  return <section className="panel-section"><div className="section-heading"><div><p className="eyebrow">nutka / dostępność</p><HelpHeading title="Tygodniowy plan" help={help.weeklyPlan} /></div><span className="badge-with-help"><span className="timezone-badge">{timezone}</span><Tooltip align="end" label="Co oznacza strefa czasowa?" text={help.timezone} /></span></div><p className="supporting-copy">Każda zmiana najpierw pokazuje skutki. W najbliższych {policy.booking_horizon_days} dniach musisz rozwiązać każdą kolizję.</p><ApiFeedback error={error} /><RuleCreate policy={policy} onPreview={(value) => void onPreview(value)} /><RuleList rules={data.availability_rules} onPreview={onPreview} /></section>;
}

function RuleList({ rules, onPreview }: { rules: CalendarResponse["availability_rules"]; onPreview: (value: AvailabilityProposal) => Promise<void> }) {
  const [editing, setEditing] = useState<string | null>(null);
  if (!rules.length) return <EmptyState>Brak reguł. Tydzień jest domyślnie niedostępny.</EmptyState>;
  return <div className="rule-list">{rules.map((rule) => <article className="rule-row" key={rule.id}>
    <div><strong>{weekdayLabels[rule.weekday]}</strong><span>{rule.start_time}–{rule.end_time} · {rule.enabled ? "włączona" : "wyłączona"}</span></div>
    {editing === rule.id
      ? <RuleEdit rule={rule} onCancel={() => setEditing(null)} onSubmit={(value) => { setEditing(null); void onPreview(value); }} />
      : <div className="row-actions"><button className="text-button" onClick={() => void onPreview({ operation: rule.enabled ? "disable" : "enable", target: "recurring_rule", id: rule.id })}>{rule.enabled ? "Wyłącz" : "Włącz"}</button><button className="text-button" onClick={() => setEditing(rule.id)}>Edytuj</button><button className="text-button danger-button" onClick={() => void onPreview({ operation: "delete", target: "recurring_rule", id: rule.id })}>Usuń</button></div>}
  </article>)}</div>;
}

function RuleEdit({ rule, onCancel, onSubmit }: { rule: CalendarResponse["availability_rules"][number]; onCancel: () => void; onSubmit: (value: AvailabilityProposal) => void }) {
  const [start, setStart] = useState(rule.start_time);
  const [end, setEnd] = useState(rule.end_time);
  return <div className="exception-edit">
    <label htmlFor={`rule-start-${rule.id}`}>Od</label>
    <input id={`rule-start-${rule.id}`} type="time" value={start} onChange={(event) => setStart(event.target.value)} />
    <label htmlFor={`rule-end-${rule.id}`}>Do</label>
    <input id={`rule-end-${rule.id}`} type="time" value={end} onChange={(event) => setEnd(event.target.value)} />
    <button className="btn btn-ghost btn-sm" onClick={() => onSubmit({ operation: "update", target: "recurring_rule", id: rule.id, rule: { start_time: start, end_time: end } })}>Sprawdź zmianę</button>
    <button className="text-button" onClick={onCancel}>Anuluj</button>
  </div>;
}

function ExceptionPanel({ data, policy, onPreview }: { data: CalendarResponse; policy: Policy; onPreview: (value: AvailabilityProposal) => Promise<void> }) {
  return <section className="panel-section"><HelpHeading title="Wyjątki dat" help={availabilityHelpCopy(policy).exceptions} /><ExceptionCreate policy={policy} onPreview={(value) => void onPreview(value)} /><ExceptionList values={data.availability_exceptions} onPreview={onPreview} /></section>;
}

function ExceptionList({ values, onPreview }: { values: CalendarResponse["availability_exceptions"]; onPreview: (value: AvailabilityProposal) => Promise<void> }) {
  const [editing, setEditing] = useState<string | null>(null);
  if (!values.length) return <EmptyState>Brak wyjątków.</EmptyState>;
  return <div className="exception-list">{values.map((item) => <article className="exception-row" key={item.id}>
    <div><strong>{item.kind === "available" ? "Dostępny" : "Niedostępny"}</strong><p>{item.start_at}–{item.end_at} · {item.enabled ? "włączony" : "wyłączony"}{item.note ? ` · ${item.note}` : ""}</p></div>
    {editing === item.id
      ? <ExceptionEdit item={item} onCancel={() => setEditing(null)} onSubmit={(value) => { setEditing(null); void onPreview(value); }} />
      : <div className="row-actions"><button className="text-button" onClick={() => void onPreview({ operation: item.enabled ? "disable" : "enable", target: "exception", id: item.id })}>{item.enabled ? "Wyłącz" : "Włącz"}</button><button className="text-button" onClick={() => setEditing(item.id)}>Edytuj</button><button className="text-button danger-button" onClick={() => void onPreview({ operation: "delete", target: "exception", id: item.id })}>Usuń</button></div>}
  </article>)}</div>;
}

function ExceptionEdit({ item, onCancel, onSubmit }: { item: CalendarResponse["availability_exceptions"][number]; onCancel: () => void; onSubmit: (value: AvailabilityProposal) => void }) {
  const [note, setNote] = useState(item.note || "");
  return <div className="exception-edit">
    <label htmlFor={`note-${item.id}`}>Notatka</label>
    <input id={`note-${item.id}`} value={note} onChange={(event) => setNote(event.target.value)} />
    <button className="btn btn-ghost btn-sm" onClick={() => onSubmit({ operation: "update", target: "exception", id: item.id, exception: { note } })}>Sprawdź zmianę</button>
    <button className="text-button" onClick={onCancel}>Anuluj</button>
  </div>;
}

function AvailabilityLessons({ data, policy }: { data: CalendarResponse; policy: Policy }) {
  const help = availabilityHelpCopy(policy);
  return <><section className="panel-section"><HelpHeading title={`Lekcje w ${policy.booking_horizon_days} dniach`} help={help.nearTerm} /><LessonList lessons={data.near_term_lessons} role="teacher" policy={policy} commercialSummaries={data.commercial_summaries} assignments={data.assignments} /></section><section className="panel-section"><HelpHeading title="Dalsze stałe rezerwacje" help={help.laterContract} /><LessonList lessons={data.later_contract_lessons ?? []} role="teacher" policy={policy} commercialSummaries={data.commercial_summaries} assignments={data.assignments} /></section></>;
}

function RuleCreate({ policy, onPreview }: { policy: Policy; onPreview: (value: AvailabilityProposal) => void }) {
  const [weekday, setWeekday] = useState("1");
  const [start, setStart] = useState("16:00");
  const [end, setEnd] = useState("20:00");
  function submit(event: FormEvent) { event.preventDefault(); onPreview({ operation: "create", target: "recurring_rule", rule: { weekday: Number(weekday), start_time: start, end_time: end, enabled: true } }); }
  return <form className="availability-form" onSubmit={submit}><label>Dzień<select value={weekday} onChange={(event) => setWeekday(event.target.value)}>{weekdayLabels.map((label, index) => <option key={label} value={index}>{label}</option>)}</select></label><label>Od<input type="time" step={policy.start_grid_minutes * 60} value={start} onChange={(event) => setStart(event.target.value)} /></label><label>Do<input type="time" step={policy.start_grid_minutes * 60} value={end} onChange={(event) => setEnd(event.target.value)} /></label><button className="btn btn-primary btn-sm" type="submit">Sprawdź regułę</button></form>;
}

function ExceptionCreate({ policy, onPreview }: { policy: Policy; onPreview: (value: AvailabilityProposal) => void }) {
  const draft = futureExceptionDraft(new Date(), policy.lesson_duration_minutes);
  const [start, setStart] = useState(draft.start);
  const [end, setEnd] = useState(draft.end);
  const [kind, setKind] = useState<"available" | "unavailable">("unavailable");
  const [note, setNote] = useState("");
  function submit(event: FormEvent) { event.preventDefault(); onPreview({ operation: "create", target: "exception", exception: { start_at: localInputToUtc(start), end_at: localInputToUtc(end), kind, note, enabled: true } }); }
  return <form className="exception-form" onSubmit={submit}><label>Od<input type="datetime-local" step={policy.start_grid_minutes * 60} value={start} onChange={(event) => setStart(event.target.value)} /></label><label>Do<input type="datetime-local" step={policy.start_grid_minutes * 60} value={end} onChange={(event) => setEnd(event.target.value)} /></label><label>Rodzaj<select value={kind} onChange={(event) => setKind(event.target.value as typeof kind)}><option value="unavailable">Niedostępny</option><option value="available">Dostępny</option></select></label><label>Notatka<input value={note} onChange={(event) => setNote(event.target.value)} /></label><button className="btn btn-primary btn-sm" type="submit">Sprawdź wyjątek</button></form>;
}

type ReviewDialogProps = { preview: AvailabilityPreview; horizonDays: number; resolutions: Record<string, AvailabilityConflictResolution>; onResolution: (value: AvailabilityConflictResolution) => void; onSave: () => void; onClose: () => void; busy: boolean };

function ReviewDialog({ preview, horizonDays, resolutions, onResolution, onSave, onClose, busy }: ReviewDialogProps) {
  return <ConfirmDialog open title="Skutki zmiany dostępności" consequence="Zapiszemy wszystkie poniższe zmiany razem albo żadnej." confirmLabel="Zapisz zmiany" busy={busy} confirmDisabled={!complete(preview, resolutions)} onConfirm={onSave} onCancel={onClose}>
    <h3>Lekcje w najbliższych {horizonDays} dniach</h3>
    {preview.near_term_conflicts.length ? preview.near_term_conflicts.map((conflict) => <ConflictRow key={conflict.lesson} lesson={conflict.lesson} startAt={conflict.start_at} value={resolutions[conflict.lesson]} onChange={onResolution} />) : <p className="supporting-copy">Zmiana nie koliduje z żadną lekcją.</p>}
    <h3>Dalsze terminy</h3>
    {preview.distant_effects.length ? <ul className="history-list">{preview.distant_effects.map((effect) => <li key={effect.occurrence}>{formatScheduleInstant(effect.start_at)} · {effect.effect === "omit" ? "lekcja zostanie pominięta" : "lekcja wróci do kalendarza"}</li>)}</ul> : <p className="supporting-copy">Dalsze terminy się nie zmienią.</p>}
  </ConfirmDialog>;
}

function ConflictRow({ lesson, startAt, value, onChange }: { lesson: string; startAt: string; value?: AvailabilityConflictResolution; onChange: (value: AvailabilityConflictResolution) => void }) {
  const when = formatScheduleInstant(startAt);
  return <div className="resolution-row">
    <p>Lekcja {when} koliduje ze zmianą.</p>
    <select aria-label={`Co zrobić z lekcją ${when}`} value={value?.action ?? ""} onChange={(event) => onChange({ lesson, action: event.target.value as "cancel" | "reschedule" })}><option value="" disabled>Wybierz, co zrobić</option><option value="cancel">Odwołaj</option><option value="reschedule">Przełóż</option></select>
    {value?.action === "reschedule" ? <input aria-label="Nowy termin" type="datetime-local" onChange={(event) => onChange({ ...value, replacement_start_at: localInputToUtc(event.target.value) })} /> : null}
  </div>;
}

function complete(preview: AvailabilityPreview, values: Record<string, AvailabilityConflictResolution>): boolean {
  return preview.near_term_conflicts.every((item) => values[item.lesson] && (values[item.lesson].action === "cancel" || Boolean(values[item.lesson].replacement_start_at)));
}
