// Lists the immutable business events of one learner, marking each correction and its internal note.

import { useQuery } from "@tanstack/react-query";

import { polishEvent } from "../../../api/copy";
import { ApiFeedback } from "../../../components/api-feedback";
import { EmptyState } from "../../../components/empty-state";
import { Skeleton } from "../../../components/skeleton";
import { historyQuery } from "../../../query/commercial";
import { formatScheduleInstant } from "../../../time/schedule";

export function HistoryTab({ accountId, assignmentId }: { accountId: string; assignmentId: string }) {
  const history = useQuery(historyQuery("teacher", accountId, assignmentId));
  if (history.error) return <ApiFeedback error={history.error} onRetry={() => void history.refetch()} />;
  if (history.isPending) return <Skeleton lines={6} label="Ładowanie historii…" />;
  const items = history.data?.items ?? [];
  if (items.length === 0) return <EmptyState>Ten uczeń nie ma jeszcze zdarzeń.</EmptyState>;
  return <ul className="history-list">{items.map((event) => <li key={event.id}><strong>{polishEvent(event.event_type)}</strong> · {formatScheduleInstant(event.event_at)}{event.corrects_event ? " · korekta" : ""}{event.internal_note ? ` · ${event.internal_note}` : ""}</li>)}</ul>;
}
