// Turns the learner payment counters into plain Polish sentences and drops every zero value.

import type { PaymentSummary } from "../../../api/contracts";
import { formatMoney } from "../../../money";

export function paymentFacts(payments: PaymentSummary): string[] {
  const facts: string[] = [];
  if (payments.overdue > 0) facts.push(`Płatności po terminie: ${payments.overdue}.`);
  if (payments.pending > 0) facts.push(`Płatności do rozliczenia: ${payments.pending}.`);
  if (payments.intentionally_unpaid > 0) facts.push(`Lekcje bez opłaty: ${payments.intentionally_unpaid}.`);
  if (payments.credit_minor > 0) facts.push(`Nadpłata do wykorzystania: ${formatMoney(payments.credit_minor, payments.currency)}.`);
  return facts;
}
