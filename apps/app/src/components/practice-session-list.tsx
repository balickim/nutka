// Lists recorded practice sessions newest first for both personas. The server marks a learner session deletable within 7 days.

import { useState } from "react";

import { deletePracticeSession, type PracticeSession } from "../api/practice";
import { ActionButton } from "./action-button";
import { ApiFeedback } from "./api-feedback";
import { EmptyState } from "./empty-state";
import { useToast } from "./toast";
import { usePracticeMutation } from "../query/practice";
import { formatLocalDate } from "../time/schedule";

export function SessionList({ assignmentId, sessions, empty }: { assignmentId: string; sessions: PracticeSession[]; empty: string }) {
  const remove = usePracticeMutation();
  const { notify } = useToast();
  const [removing, setRemoving] = useState<string | null>(null);
  async function removeSession(session: PracticeSession) {
    setRemoving(session.id);
    await remove.mutateAsync({ assignmentId, write: () => deletePracticeSession(session.id) });
    notify("Usunięto wpis.");
  }
  if (sessions.length === 0) return <EmptyState>{empty}</EmptyState>;
  return <>
    <ApiFeedback error={remove.error} />
    <ul className="session-list">{sessions.map((session) => <li key={session.id}>
      <div><strong>{formatLocalDate(session.practiced_on)}</strong>{session.minutes ? ` · ${session.minutes} min` : ""}
        {session.tasks.length ? <p className="lesson-meta">{session.tasks.map((task) => task.title).join(", ")}</p> : null}
        {session.comment ? <p className="session-comment">„{session.comment}”</p> : null}
      </div>
      {session.deletable ? <ActionButton variant="text" busy={remove.isPending && removing === session.id} onClick={() => void removeSession(session).catch(() => undefined)}>Usuń</ActionButton> : null}
    </li>)}</ul>
  </>;
}
