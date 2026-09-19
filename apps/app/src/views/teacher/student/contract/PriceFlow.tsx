// Sets a future lesson price in major units, leaving settled months untouched.

import { useState } from "react";

import { amendContractPrice } from "../../../../api/commercial";
import type { RegularContract } from "../../../../api/contracts";
import { FlowPanel } from "../../../../components/FlowPanel";
import { useToast } from "../../../../components/Toast";
import { formatMoney, parseMajor, toMajorInput } from "../../../../money";
import { usePlanMutation } from "../../../../query/commercial";
import { formatScheduleDate } from "../../../../time/schedule";

export function PriceFlow({ assignmentId, contract }: { assignmentId: string; contract: RegularContract }) {
  const mutation = usePlanMutation();
  const { notify } = useToast();
  const [effectiveOn, setEffectiveOn] = useState("");
  const [price, setPrice] = useState(toMajorInput(contract.price_minor));
  const minor = parseMajor(price);
  const summary = summaryOf(effectiveOn, minor, contract.currency);
  async function confirm() {
    await mutation.mutateAsync({ assignmentId, write: () => amendContractPrice(contract.id, { effective_on: effectiveOn, price_minor: minor ?? 0, currency: contract.currency }) });
    notify("Zapisano przyszłą cenę.");
  }
  return <FlowPanel title="Zmiana ceny" openLabel="Zmień cenę" confirmLabel="Zapisz cenę" summary={summary} busy={mutation.isPending} onConfirm={confirm}>
    <label htmlFor={`price-on-${contract.id}`}>Obowiązuje od</label>
    <input id={`price-on-${contract.id}`} type="date" value={effectiveOn} onChange={(event) => setEffectiveOn(event.target.value)} required />
    <label htmlFor={`price-${contract.id}`}>Cena za lekcję w {contract.currency}</label>
    <input id={`price-${contract.id}`} inputMode="decimal" value={price} onChange={(event) => setPrice(event.target.value)} aria-invalid={minor === null} />
    {minor === null ? <p className="field-error">Podaj kwotę, na przykład 50,00.</p> : null}
  </FlowPanel>;
}

function summaryOf(effectiveOn: string, minor: number | null, currency: string): string | null {
  if (!effectiveOn || minor === null) return null;
  return `Od ${formatScheduleDate(`${effectiveOn}T12:00:00Z`)} lekcja kosztuje ${formatMoney(minor, currency)}. Wcześniejsze rozliczenia zostają bez zmian.`;
}
