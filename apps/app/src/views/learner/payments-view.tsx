// Tells the learner how much to pay, by when, and how: open items, transfer details with copy controls and a QR code, forecast, and recent payments.

import { useQuery } from "@tanstack/react-query";

import { lessonCount } from "../../api/copy";
import type { DueItem, PaymentDetails, PaymentDue, RecentPayment } from "../../api/payments";
import { ApiFeedback } from "../../components/api-feedback";
import { EmptyState } from "../../components/empty-state";
import { Skeleton } from "../../components/skeleton";
import { useToast } from "../../components/toast";
import { formatMoney } from "../../money";
import { paymentDueQuery } from "../../query/payments";
import { formatLocalDate, formatScheduleDate } from "../../time/schedule";
import { LearnerShell, type LearnerContext } from "./learner-shell";
import { itemLabel, monthName, transferTitle, zbpPayload } from "./payments/transfer";
import { TransferQr } from "./payments/transfer-qr";

export function PaymentsView() {
  return <LearnerShell lede="Ile zapłacić, do kiedy i na jakie konto.">{(context) => <Payments context={context} />}</LearnerShell>;
}

function Payments({ context }: { context: LearnerContext }) {
  const due = useQuery(paymentDueQuery(context.accountId, context.assignment.id));
  if (due.error) return <ApiFeedback error={due.error} onRetry={() => void due.refetch()} />;
  if (!due.data) return <Skeleton lines={5} label="Ładowanie płatności…" />;
  const title = transferTitle(context.assignment.learner_name ?? "", due.data.items);
  return <>
    <DueSection due={due.data} />
    {due.data.total_minor > 0 ? <HowToPay instructions={due.data.instructions} amountMinor={due.data.total_minor} title={title} /> : null}
    <ForecastSection due={due.data} />
    <RecentSection payments={due.data.recent_payments} />
  </>;
}

function DueSection({ due }: { due: PaymentDue }) {
  return <section className="panel-section">
    <h2>{due.total_minor > 0 ? `Do zapłaty: ${formatMoney(due.total_minor, due.currency)}` : "Wszystko opłacone"}</h2>
    {due.total_minor > 0 ? <ul className="due-list">{due.items.map((item) => <DueRow key={item.charge} item={item} />)}</ul> : <EmptyState>Nie masz teraz nic do zapłaty.</EmptyState>}
    {due.open_credit_minor > 0 ? <p className="supporting-copy">Nadpłata {formatMoney(due.open_credit_minor, due.currency)} zmniejszy kolejną opłatę.</p> : null}
    <p className="supporting-copy">Nauczyciel zapisuje wpłaty ręcznie, więc świeży przelew może jeszcze chwilę widnieć jako do zapłaty.</p>
  </section>;
}

function DueRow({ item }: { item: DueItem }) {
  return <li className="due-row"><div><strong>{describe(item)}</strong>{item.due_on ? <p className="supporting-copy">Termin: {formatLocalDate(item.due_on)}</p> : null}</div>
    <div className="due-amount">{formatMoney(item.amount_minor, "PLN")}{item.overdue ? <span className="status-badge status-overdue">Po terminie</span> : null}</div></li>;
}

function HowToPay({ instructions, amountMinor, title }: { instructions: PaymentDetails | null; amountMinor: number; title: string }) {
  if (!instructions) return <section className="panel-section"><h2>Jak zapłacić</h2><EmptyState>Zapytaj nauczyciela o sposób płatności.</EmptyState></section>;
  const payload = zbpPayload(instructions.iban, instructions.account_holder, amountMinor, title);
  return <section className="panel-section"><h2>Jak zapłacić</h2>
    <div className="transfer-details">
      <dl className="transfer-fields">
        <dt>Odbiorca</dt><dd>{instructions.account_holder}</dd>
        <dt>Numer konta</dt><dd><span className="iban">{groupIban(instructions.iban)}</span> <CopyButton value={instructions.iban.slice(2)} label="Kopiuj numer konta" done="Skopiowano numer konta." /></dd>
        <dt>Tytuł przelewu</dt><dd>{title} <CopyButton value={title} label="Kopiuj tytuł" done="Skopiowano tytuł przelewu." /></dd>
        <dt>Kwota</dt><dd>{formatMoney(amountMinor, "PLN")}</dd>
      </dl>
      {payload ? <TransferQr payload={payload} /> : null}
    </div>
    {instructions.note ? <p className="supporting-copy">{instructions.note}</p> : null}
  </section>;
}

function CopyButton({ value, label, done }: { value: string; label: string; done: string }) {
  const { notify } = useToast();
  return <button type="button" className="btn btn-ghost btn-sm" onClick={() => void navigator.clipboard.writeText(value).then(() => notify(done), () => notify("Nie udało się skopiować. Zaznacz tekst ręcznie."))}>{label}</button>;
}

function ForecastSection({ due }: { due: PaymentDue }) {
  const forecast = due.next_forecast;
  if (!forecast) return null;
  return <section className="panel-section"><h2>Następny miesiąc</h2><p>{capitalize(monthName(forecast.month))}: {lessonCount(forecast.lesson_count)}, {formatMoney(forecast.amount_minor, due.currency)}. Termin płatności: {formatLocalDate(forecast.due_on)}.</p></section>;
}

function RecentSection({ payments }: { payments: RecentPayment[] }) {
  if (payments.length === 0) return null;
  return <section className="panel-section"><h2>Ostatnie wpłaty</h2><ul className="due-list">{payments.map((payment) => <li className="due-row" key={payment.charge}><div><strong>{describe(payment)}</strong><p className="supporting-copy">Zapisano {formatScheduleDate(payment.paid_at)}</p></div><div className="due-amount">{formatMoney(payment.amount_minor, "PLN")}</div></li>)}</ul></section>;
}

function describe(item: Pick<DueItem, "kind" | "period" | "lesson_start_at">): string {
  return item.kind === "lesson" ? `Lekcja ${itemLabel(item)}` : `Lekcje, ${monthName(item.period ?? "")}`;
}

function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function groupIban(iban: string): string {
  return iban.slice(2).replace(/(\d{2})(\d{4})(\d{4})(\d{4})(\d{4})(\d{4})(\d{4})/, "$1 $2 $3 $4 $5 $6 $7");
}
