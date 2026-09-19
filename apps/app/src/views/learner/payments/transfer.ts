// Builds the bank transfer title and the ZBP two-dimensional transfer code payload from the payment-due read.

import type { DueItem } from "../../../api/payments";
import { localizeUtcInstant } from "../../../time/utc";

const titleMaxLength = 32;
const holderMaxLength = 20;
const maxQrAmountMinor = 999_999;

export function itemLabel(item: Pick<DueItem, "kind" | "period" | "lesson_start_at">): string {
  if (item.kind === "lesson" && item.lesson_start_at) return localizeUtcInstant(item.lesson_start_at, "pl-PL", { day: "numeric", month: "numeric", year: "numeric" });
  const [year, month] = (item.period ?? "").split("-");
  return `${month}.${year}`;
}

// A month name without a day uses the nominative form in Polish, for example "październik 2026".
export function monthName(period: string): string {
  const [year, month] = period.split("-").map(Number);
  return new Intl.DateTimeFormat("pl-PL", { month: "long", year: "numeric", timeZone: "UTC" }).format(new Date(Date.UTC(year, month - 1, 1))).replace(/ r\.$/, "");
}

// The ZBP code allows 32 title characters, so the title names the learner and as many periods as fit.
export function transferTitle(learnerName: string, items: DueItem[]): string {
  const title = [`Lekcje ${learnerName}`.trim(), ...items.map(itemLabel)].join(" ");
  return clip(title, titleMaxLength);
}

// The payload follows the ZBP recommendation: |country|account|amount in grosze|recipient|title|||.
export function zbpPayload(iban: string, holder: string, amountMinor: number, title: string): string | null {
  if (!/^PL\d{26}$/.test(iban) || amountMinor <= 0 || amountMinor > maxQrAmountMinor) return null;
  const amount = String(amountMinor).padStart(6, "0");
  return `|PL|${iban.slice(2)}|${amount}|${clip(holder, holderMaxLength)}|${clip(title, titleMaxLength)}|||`;
}

function clip(value: string, length: number): string {
  return value.replaceAll("|", " ").slice(0, length).trim();
}
