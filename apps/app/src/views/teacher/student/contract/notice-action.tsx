// Submits a contract notice after a dialog names the resolved end date.

import { useState } from "react";

import { submitContractNotice } from "../../../../api/commercial";
import type { RegularContract } from "../../../../api/contracts";
import { ActionButton } from "../../../../components/action-button";
import { ConfirmDialog } from "../../../../components/confirm-dialog";
import { useToast } from "../../../../components/toast";
import { usePlanMutation } from "../../../../query/commercial";
import { formatScheduleDate } from "../../../../time/schedule";

export function NoticeAction({ assignmentId, contract }: { assignmentId: string; contract: RegularContract }) {
  const mutation = usePlanMutation();
  const { notify } = useToast();
  const [open, setOpen] = useState(false);
  async function confirm() {
    await mutation.mutateAsync({ assignmentId, write: () => submitContractNotice("teacher", contract.id) });
    notify("Wypowiedzenie złożone.");
    setOpen(false);
  }
  return <>
    <ActionButton danger variant="text" onClick={() => setOpen(true)}>Wypowiedz umowę</ActionButton>
    <ConfirmDialog open={open} title="Wypowiedzenie umowy" consequence={`Umowa zakończy się ${formatScheduleDate(`${contract.effective_end_on || contract.end_on}T12:00:00Z`)}. Lekcje po tej dacie znikną z kalendarza.`} confirmLabel="Złóż wypowiedzenie" danger busy={mutation.isPending} onConfirm={() => void confirm()} onCancel={() => setOpen(false)} />
  </>;
}
