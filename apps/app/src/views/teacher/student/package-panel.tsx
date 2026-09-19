// Runs package purchase and renewal as guided flows, and keeps closure and token correction behind advanced operations.

import { useState } from "react";

import { closePackage, correctPackage, purchasePackage } from "../../../api/commercial";
import { polishTokenState } from "../../../api/copy";
import type { Lesson, Package, Policy } from "../../../api/contracts";
import { ActionButton } from "../../../components/action-button";
import { AdvancedOperations } from "../../../components/advanced-operations";
import { ApiFeedback } from "../../../components/api-feedback";
import { FlowPanel } from "../../../components/flow-panel";
import { PolicyHint } from "../../../components/policy-hint";
import { useToast } from "../../../components/toast";
import { formatMoney, parseMajor } from "../../../money";
import { usePlanMutation, useCorrectionMutation } from "../../../query/commercial";
import { EventPicker } from "./event-picker";
import { LessonPicker } from "./lesson-picker";

export function PackagePanel({ accountId, assignmentId, packages, adHocLessons, policy }: { accountId: string; assignmentId: string; packages: Package[]; adHocLessons: Lesson[]; policy: Policy }) {
  const current = packages.find((item) => item.status === "open");
  return <section className="subpanel">
    <h3>Pakiet {policy.package_token_count} lekcji <PolicyHint>{`${formatMoney(policy.package_price_minor, policy.currency)}, ważny ${policy.package_validity_days} dni.`}</PolicyHint></h3>
    {current ? <OpenPackage accountId={accountId} assignmentId={assignmentId} value={current} adHocLessons={adHocLessons} policy={policy} /> : <PurchaseFlow assignmentId={assignmentId} adHocLessons={adHocLessons} policy={policy} renewal={false} />}
  </section>;
}

function OpenPackage({ accountId, assignmentId, value, adHocLessons, policy }: { accountId: string; assignmentId: string; value: Package; adHocLessons: Lesson[]; policy: Policy }) {
  const available = value.tokens.filter((token) => token.state === "available").length;
  return <>
    <p>Ważny do {value.valid_through}. Dostępne lekcje: {available} z {value.tokens.length}. Cena {formatMoney(value.price_minor, value.currency)}.</p>
    <ul className="history-list">{value.tokens.map((token) => <li key={token.id}>#{token.ordinal}: {polishTokenState(token.state)}</li>)}</ul>
    {available === 0 ? <PurchaseFlow assignmentId={assignmentId} adHocLessons={adHocLessons} policy={policy} renewal /> : null}
    <PackageCorrections accountId={accountId} assignmentId={assignmentId} value={value} />
  </>;
}

function PurchaseFlow({ assignmentId, adHocLessons, policy, renewal }: { assignmentId: string; adHocLessons: Lesson[]; policy: Policy; renewal: boolean }) {
  const mutation = usePlanMutation();
  const { notify } = useToast();
  const [convert, setConvert] = useState<string[]>([]);
  const title = renewal ? "Odnowienie pakietu" : "Zakup pakietu";
  const summary = `${title}: ${policy.package_token_count} lekcji za ${formatMoney(policy.package_price_minor, policy.currency)}, ważny ${policy.package_validity_days} dni.${convert.length ? ` Zamieniamy ${convert.length} pojedynczych lekcji na lekcje z pakietu.` : ""}`;
  async function confirm() {
    await mutation.mutateAsync({ assignmentId, write: () => purchasePackage(assignmentId, convert.length ? { convert_lesson_ids: convert } : {}) });
    notify(`${title} zapisane.`);
    setConvert([]);
  }
  return <>
    <ApiFeedback error={mutation.error} />
    <FlowPanel title={title} openLabel={renewal ? "Odnów pakiet" : "Oznacz zakup pakietu"} confirmLabel={renewal ? "Odnów pakiet" : "Zapisz zakup"} summary={summary} busy={mutation.isPending} onConfirm={confirm}>
      <LessonPicker lessons={adHocLessons} selected={convert} onChange={setConvert} />
    </FlowPanel>
  </>;
}

function PackageCorrections({ accountId, assignmentId, value }: { accountId: string; assignmentId: string; value: Package }) {
  const plan = usePlanMutation();
  const correction = useCorrectionMutation();
  const { notify } = useToast();
  const [refund, setRefund] = useState("0,00");
  const [tokenId, setTokenId] = useState("");
  const [eventId, setEventId] = useState("");
  const amount = parseMajor(refund);
  const usedTokens = value.tokens.filter((token) => token.state !== "available");
  return <AdvancedOperations>
    {(reason) => <>
      <ApiFeedback error={plan.error || correction.error} />
      <label htmlFor={`refund-${value.id}`}>Kwota zwrotu w {value.currency}, opcjonalna</label>
      <input id={`refund-${value.id}`} inputMode="decimal" value={refund} onChange={(event) => setRefund(event.target.value)} aria-invalid={amount === null} />
      <ActionButton danger busy={plan.isPending} disabled={!reason.trim() || amount === null} onClick={() => void closeIt(reason)}>Zamknij pakiet</ActionButton>
      <label htmlFor={`token-${value.id}`}>Lekcja pakietu do przywrócenia</label>
      <select id={`token-${value.id}`} value={tokenId} onChange={(event) => setTokenId(event.target.value)}>
        <option value="">Wybierz lekcję</option>
        {usedTokens.map((token) => <option key={token.id} value={token.id}>#{token.ordinal} · {polishTokenState(token.state)}</option>)}
      </select>
      <EventPicker accountId={accountId} assignmentId={assignmentId} value={eventId} onChange={setEventId} />
      <ActionButton busy={correction.isPending} disabled={!reason.trim() || !tokenId || !eventId} onClick={() => void correct(reason)}>Przywróć lekcję pakietu</ActionButton>
    </>}
  </AdvancedOperations>;

  async function closeIt(reason: string) {
    await plan.mutateAsync({ assignmentId, write: () => closePackage(value.id, { reason, ...(amount && amount > 0 ? { refund: { amount_minor: amount, currency: value.currency } } : {}) }) });
    notify("Pakiet zamknięty. Zapisano korektę w historii.");
  }
  async function correct(reason: string) {
    await correction.mutateAsync({ assignmentId, write: () => correctPackage(value.id, { token_id: tokenId, target: "available", corrects_event: eventId, reason }) });
    notify("Zapisano korektę lekcji pakietu.");
  }
}
