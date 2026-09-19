// Selects the event a correction targets from the learner's history, so no flow asks for an event identifier.

import { useQuery } from "@tanstack/react-query";

import { polishEvent } from "../../../api/copy";
import { historyQuery } from "../../../query/commercial";
import { formatScheduleInstant } from "../../../time/schedule";

export function EventPicker({ accountId, assignmentId, value, onChange }: { accountId: string; assignmentId: string; value: string; onChange: (value: string) => void }) {
  const history = useQuery(historyQuery("teacher", accountId, assignmentId));
  const items = history.data?.items ?? [];
  return <label>Zdarzenie do korekty
    <select value={value} onChange={(event) => onChange(event.target.value)}>
      <option value="">Wybierz zdarzenie</option>
      {items.map((event) => <option key={event.id} value={event.id}>{polishEvent(event.event_type)} · {formatScheduleInstant(event.event_at)}</option>)}
    </select>
  </label>;
}
