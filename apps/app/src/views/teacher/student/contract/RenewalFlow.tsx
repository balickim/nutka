// Starts a new contract term on the same conditions from a chosen date.

import { useState } from "react";

import { renewContract } from "../../../../api/commercial";
import type { RegularContract } from "../../../../api/contracts";
import { FlowPanel } from "../../../../components/FlowPanel";
import { useToast } from "../../../../components/Toast";
import { usePlanMutation } from "../../../../query/commercial";
import { formatScheduleDate } from "../../../../time/schedule";

export function RenewalFlow({ assignmentId, contract }: { assignmentId: string; contract: RegularContract }) {
  const mutation = usePlanMutation();
  const { notify } = useToast();
  const [startOn, setStartOn] = useState("");
  const summary = startOn ? `Nowa umowa zacznie się ${formatScheduleDate(`${startOn}T12:00:00Z`)} na tych samych warunkach.` : null;
  async function confirm() {
    await mutation.mutateAsync({ assignmentId, write: () => renewContract(contract.id, { start_on: startOn }) });
    notify("Umowa odnowiona.");
  }
  return <FlowPanel title="Odnowienie umowy" openLabel="Odnów umowę" confirmLabel="Odnów umowę" summary={summary} busy={mutation.isPending} onConfirm={confirm}>
    <label htmlFor={`renew-${contract.id}`}>Data początku</label>
    <input id={`renew-${contract.id}`} type="date" value={startOn} onChange={(event) => setStartOn(event.target.value)} required />
  </FlowPanel>;
}
