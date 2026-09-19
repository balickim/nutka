// Composes the flows and corrections available on an active weekly contract.

import type { Policy, RegularContract } from "../../../api/contracts";
import { formatMoney } from "../../../money";
import { weekdayLabels } from "../../../time/schedule";
import { ContractCorrections } from "./contract/ContractCorrections";
import { ContractSeries } from "./contract/ContractSeries";
import { NoticeAction } from "./contract/NoticeAction";
import { PriceFlow } from "./contract/PriceFlow";
import { RenewalFlow } from "./contract/RenewalFlow";
import { ScheduleChangeFlow } from "./contract/ScheduleChangeFlow";

export function ContractActions({ accountId, assignmentId, contract, policy }: { accountId: string; assignmentId: string; contract: RegularContract; policy: Policy }) {
  return <>
    <p>{contract.start_on}–{contract.effective_end_on || contract.end_on} · {weekdayLabels[contract.weekday]} {contract.start_time} · {formatMoney(contract.price_minor, contract.currency)}</p>
    <p className="supporting-copy">Pozostałe przełożenia: {contract.remaining_monthly_reschedules}. Bezpłatne odwołania: {contract.remaining_free_cancellations}.</p>
    <ScheduleChangeFlow assignmentId={assignmentId} contract={contract} policy={policy} />
    <NoticeAction assignmentId={assignmentId} contract={contract} />
    <RenewalFlow assignmentId={assignmentId} contract={contract} />
    <PriceFlow assignmentId={assignmentId} contract={contract} />
    <ContractSeries assignmentId={assignmentId} contract={contract} />
    <ContractCorrections accountId={accountId} assignmentId={assignmentId} contract={contract} />
  </>;
}
