// Edits teacher availability only through impact preview, explicit conflict resolutions, and one atomic commit.

import { useState, type FormEvent } from "react";

import { commitAvailability, previewAvailability } from "../../api/commercial";
import type { AvailabilityConflictResolution, AvailabilityPreview, AvailabilityProposal, CalendarResponse, Policy } from "../../api/contracts";
import { ApiFeedback, EmptyState, LessonList } from "../../components/ScheduleBits";
import { useAvailabilityCommitMutation } from "../../query/commercial";
import { futureExceptionDraft, localInputToUtc, weekdayLabels } from "../../time/schedule";

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
    {preview && proposal ? <PreviewPanel preview={preview} resolutions={resolutions} onResolution={(value) => setResolutions((current) => ({ ...current, [value.lesson]: value }))} onSave={() => void save()} onClose={() => setPreview(null)} busy={commit.isPending} /> : null}
    <AvailabilityLessons data={data} policy={policy} />
  </>;
}

function RulePanel({ data, policy, timezone, onPreview, error }: { data: CalendarResponse; policy: Policy; timezone: string; onPreview: (value: AvailabilityProposal) => Promise<void>; error: unknown }) {
  return <section className="panel-section"><div className="section-heading"><div><p className="eyebrow">nutka / dostępność</p><h2>Tygodniowy plan</h2></div><span className="timezone-badge">{timezone}</span></div><p className="supporting-copy">Każda zmiana najpierw pokazuje skutki. W najbliższych {policy.booking_horizon_days} dniach musisz rozwiązać każdą kolizję.</p><ApiFeedback error={error} /><RuleCreate policy={policy} onPreview={(value) => void onPreview(value)} /><RuleList rules={data.availability_rules} onPreview={onPreview} /></section>;
}

function RuleList({ rules, onPreview }: { rules: CalendarResponse["availability_rules"]; onPreview: (value: AvailabilityProposal) => Promise<void> }) {
  if (!rules.length) return <EmptyState>Brak reguł. Tydzień jest domyślnie niedostępny.</EmptyState>;
  return <div className="rule-list">{rules.map((rule) => <article className="rule-row" key={rule.id}><div><strong>{weekdayLabels[rule.weekday]}</strong><span>{rule.start_time}–{rule.end_time} · {rule.enabled ? "włączona" : "wyłączona"}</span></div><div className="row-actions"><button className="text-button" onClick={() => void onPreview({ operation: rule.enabled ? "disable" : "enable", target: "recurring_rule", id: rule.id })}>{rule.enabled ? "Wyłącz" : "Włącz"}</button><button className="text-button" onClick={() => void onPreview(editRule(rule))}>Edytuj</button><button className="text-button danger-button" onClick={() => void onPreview({ operation: "delete", target: "recurring_rule", id: rule.id })}>Usuń</button></div></article>)}</div>;
}

function ExceptionPanel({ data, policy, onPreview }: { data: CalendarResponse; policy: Policy; onPreview: (value: AvailabilityProposal) => Promise<void> }) {
  return <section className="panel-section"><h2>Wyjątki dat</h2><ExceptionCreate policy={policy} onPreview={(value) => void onPreview(value)} /><ExceptionList values={data.availability_exceptions} onPreview={onPreview} /></section>;
}

function ExceptionList({ values, onPreview }: { values: CalendarResponse["availability_exceptions"]; onPreview: (value: AvailabilityProposal) => Promise<void> }) {
  if (!values.length) return <EmptyState>Brak wyjątków.</EmptyState>;
  return <div className="exception-list">{values.map((item) => <article className="exception-row" key={item.id}><div><strong>{item.kind === "available" ? "Dostępny" : "Niedostępny"}</strong><p>{item.start_at}–{item.end_at} · {item.enabled ? "włączony" : "wyłączony"}{item.note ? ` · ${item.note}` : ""}</p></div><div className="row-actions"><button className="text-button" onClick={() => void onPreview({ operation: item.enabled ? "disable" : "enable", target: "exception", id: item.id })}>{item.enabled ? "Wyłącz" : "Włącz"}</button><button className="text-button" onClick={() => void onPreview(editException(item))}>Edytuj</button><button className="text-button danger-button" onClick={() => void onPreview({ operation: "delete", target: "exception", id: item.id })}>Usuń</button></div></article>)}</div>;
}

function AvailabilityLessons({ data, policy }: { data: CalendarResponse; policy: Policy }) {
  return <><section className="panel-section"><h2>Lekcje w {policy.booking_horizon_days} dniach</h2><LessonList lessons={data.near_term_lessons} role="teacher" policy={policy} commercialSummaries={data.commercial_summaries} /></section><section className="panel-section"><h2>Dalsze stałe rezerwacje</h2><LessonList lessons={data.later_contract_lessons ?? []} role="teacher" policy={policy} commercialSummaries={data.commercial_summaries} /></section></>;
}

function RuleCreate({ policy, onPreview }: { policy: Policy; onPreview: (value: AvailabilityProposal) => void }) {
  const [weekday, setWeekday] = useState("1");
  const [start, setStart] = useState("16:00");
  const [end, setEnd] = useState("20:00");
  function submit(event: FormEvent) { event.preventDefault(); onPreview({ operation: "create", target: "recurring_rule", rule: { weekday: Number(weekday), start_time: start, end_time: end, enabled: true } }); }
  return <form className="availability-form" onSubmit={submit}><label>Dzień<select value={weekday} onChange={(event) => setWeekday(event.target.value)}>{weekdayLabels.map((label, index) => <option key={label} value={index}>{label}</option>)}</select></label><label>Od<input type="time" step={policy.start_grid_minutes * 60} value={start} onChange={(event) => setStart(event.target.value)} /></label><label>Do<input type="time" step={policy.start_grid_minutes * 60} value={end} onChange={(event) => setEnd(event.target.value)} /></label><button className="primary-button" type="submit">Sprawdź regułę</button></form>;
}

function ExceptionCreate({ policy, onPreview }: { policy: Policy; onPreview: (value: AvailabilityProposal) => void }) {
  const draft = futureExceptionDraft(new Date(), policy.lesson_duration_minutes);
  const [start, setStart] = useState(draft.start);
  const [end, setEnd] = useState(draft.end);
  const [kind, setKind] = useState<"available" | "unavailable">("unavailable");
  const [note, setNote] = useState("");
  function submit(event: FormEvent) { event.preventDefault(); onPreview({ operation: "create", target: "exception", exception: { start_at: localInputToUtc(start), end_at: localInputToUtc(end), kind, note, enabled: true } }); }
  return <form className="exception-form" onSubmit={submit}><label>Od<input type="datetime-local" step={policy.start_grid_minutes * 60} value={start} onChange={(event) => setStart(event.target.value)} /></label><label>Do<input type="datetime-local" step={policy.start_grid_minutes * 60} value={end} onChange={(event) => setEnd(event.target.value)} /></label><label>Rodzaj<select value={kind} onChange={(event) => setKind(event.target.value as typeof kind)}><option value="unavailable">Niedostępny</option><option value="available">Dostępny</option></select></label><label>Notatka<input value={note} onChange={(event) => setNote(event.target.value)} /></label><button className="primary-button" type="submit">Sprawdź wyjątek</button></form>;
}

function PreviewPanel({ preview, resolutions, onResolution, onSave, onClose, busy }: { preview: AvailabilityPreview; resolutions: Record<string, AvailabilityConflictResolution>; onResolution: (value: AvailabilityConflictResolution) => void; onSave: () => void; onClose: () => void; busy: boolean }) {
  return <section className="preview-panel" role="dialog" aria-label="Skutki zmiany dostępności"><h2>Skutki przed zapisem</h2>{preview.near_term_conflicts.length ? preview.near_term_conflicts.map((conflict) => { const current = resolutions[conflict.lesson]; return <div className="resolution-row" key={conflict.lesson}><p>Lekcja {conflict.lesson} · {conflict.start_at}</p><select aria-label={`Rozwiązanie ${conflict.lesson}`} value={current?.action ?? ""} onChange={(event) => onResolution({ lesson: conflict.lesson, action: event.target.value as "cancel" | "reschedule" })}><option value="" disabled>Wybierz rozwiązanie</option><option value="cancel">Odwołaj</option><option value="reschedule">Przełóż</option></select>{current?.action === "reschedule" ? <input aria-label="Nowy termin" type="datetime-local" onChange={(event) => onResolution({ ...current, replacement_start_at: localInputToUtc(event.target.value) })} /> : null}</div>; }) : <p>Brak kolizji w najbliższych 14 dniach.</p>}<h3>Dalsze skutki automatyczne</h3>{preview.distant_effects.length ? <ul>{preview.distant_effects.map((effect) => <li key={effect.occurrence}>{effect.effect === "omit" ? "Pomiń" : "Przywróć"} {effect.start_at}</li>)}</ul> : <p>Brak dalszych zmian.</p>}<div className="row-actions"><button className="primary-button" disabled={busy || !complete(preview, resolutions)} onClick={onSave}>Zapisz wszystko atomowo</button><button className="text-button" onClick={onClose}>Anuluj</button></div></section>;
}

function complete(preview: AvailabilityPreview, values: Record<string, AvailabilityConflictResolution>): boolean {
  return preview.near_term_conflicts.every((item) => values[item.lesson] && (values[item.lesson].action === "cancel" || Boolean(values[item.lesson].replacement_start_at)));
}

function editRule(rule: CalendarResponse["availability_rules"][number]): AvailabilityProposal {
  const start = window.prompt("Godzina początku", rule.start_time);
  const end = window.prompt("Godzina końca", rule.end_time);
  if (!start || !end) return { operation: "update", target: "recurring_rule", id: rule.id, rule: {} };
  return { operation: "update", target: "recurring_rule", id: rule.id, rule: { start_time: start, end_time: end } };
}

function editException(item: CalendarResponse["availability_exceptions"][number]): AvailabilityProposal {
  const note = window.prompt("Notatka", item.note || "");
  return { operation: "update", target: "exception", id: item.id, exception: { note: note ?? item.note } };
}
