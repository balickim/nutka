// Ends a contract early or corrects one of its events, each behind the advanced-operations disclosure.

import { useState } from "react";

import { correctContract, endContractEarly } from "../../../../api/commercial";
import type { RegularContract } from "../../../../api/contracts";
import { ActionButton } from "../../../../components/action-button";
import { AdvancedOperations } from "../../../../components/advanced-operations";
import { ApiFeedback } from "../../../../components/api-feedback";
import { useToast } from "../../../../components/toast";
import { usePlanMutation, useCorrectionMutation } from "../../../../query/commercial";
import { formatScheduleDate } from "../../../../time/schedule";
import { EventPicker } from "../event-picker";

export function ContractCorrections({ accountId, assignmentId, contract }: { accountId: string; assignmentId: string; contract: RegularContract }) {
  const plan = usePlanMutation();
  const correction = useCorrectionMutation();
  const { notify } = useToast();
  const [endOn, setEndOn] = useState("");
  const [eventId, setEventId] = useState("");
  async function end(reason: string) {
    await plan.mutateAsync({ assignmentId, write: () => endContractEarly(contract.id, { end_on: endOn, reason }) });
    notify(`Umowa kończy się ${formatScheduleDate(`${endOn}T12:00:00Z`)}. Zapisano korektę.`);
  }
  async function correct(reason: string) {
    await correction.mutateAsync({ assignmentId, write: () => correctContract(contract.id, { event_id: eventId, reason }) });
    notify("Zapisano korektę umowy.");
  }
  return <AdvancedOperations>
    {(reason) => <>
      <ApiFeedback error={plan.error || correction.error} />
      <label htmlFor={`end-${contract.id}`}>Data wcześniejszego końca</label>
      <input id={`end-${contract.id}`} type="date" value={endOn} onChange={(event) => setEndOn(event.target.value)} />
      <ActionButton danger busy={plan.isPending} disabled={!reason.trim() || !endOn} onClick={() => void end(reason)}>Zakończ umowę wcześniej</ActionButton>
      <EventPicker accountId={accountId} assignmentId={assignmentId} value={eventId} onChange={setEventId} />
      <ActionButton busy={correction.isPending} disabled={!reason.trim() || !eventId} onClick={() => void correct(reason)}>Zapisz korektę</ActionButton>
    </>}
  </AdvancedOperations>;
}
