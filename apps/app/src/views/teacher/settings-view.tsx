// Lets the teacher set the bank transfer details that the teacher's learners see on their payments screen.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import type { PaymentDetails } from "../../api/payments";
import { ActionButton } from "../../components/action-button";
import { ApiFeedback } from "../../components/api-feedback";
import { Skeleton } from "../../components/skeleton";
import { useToast } from "../../components/toast";
import { paymentDetailsQuery, usePaymentDetailsMutation } from "../../query/payments";
import { TeacherShell } from "./teacher-shell";

export function SettingsView() {
  return <TeacherShell lede="Dane, które widzą Twoi uczniowie.">{(context) => <PaymentDetailsSection accountId={context.accountId} />}</TeacherShell>;
}

function PaymentDetailsSection({ accountId }: { accountId: string }) {
  const details = useQuery(paymentDetailsQuery(accountId));
  return <section className="panel-section"><h2>Dane do przelewu</h2>
    <p className="supporting-copy">Uczniowie widzą te dane na ekranie płatności razem z kwotą, tytułem przelewu i kodem QR. Puste pola ukrywają dane do przelewu.</p>
    {details.error ? <ApiFeedback error={details.error} onRetry={() => void details.refetch()} /> : details.data ? <PaymentDetailsForm accountId={accountId} initial={details.data} /> : <Skeleton lines={3} label="Ładowanie danych…" />}
  </section>;
}

function PaymentDetailsForm({ accountId, initial }: { accountId: string; initial: PaymentDetails }) {
  const mutation = usePaymentDetailsMutation(accountId);
  const { notify } = useToast();
  const [values, setValues] = useState(initial);
  const field = (name: keyof PaymentDetails) => ({ id: `payment-${name}`, value: values[name], onChange: (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => setValues({ ...values, [name]: event.target.value }) });
  async function save(event: React.FormEvent) {
    event.preventDefault();
    const saved = await mutation.mutateAsync(values);
    setValues(saved);
    notify("Zapisano dane do przelewu.");
  }
  return <form className="inline-form" onSubmit={(event) => void save(event).catch(() => undefined)}>
    <label htmlFor="payment-account_holder">Odbiorca przelewu</label>
    <input {...field("account_holder")} maxLength={140} autoComplete="name" />
    <label htmlFor="payment-iban">Numer konta</label>
    <input {...field("iban")} inputMode="numeric" placeholder="61 1090 1014 0000 0712 1981 2874" />
    <label htmlFor="payment-note">Dodatkowa informacja (opcjonalna)</label>
    <textarea {...field("note")} maxLength={300} rows={2} placeholder="Na przykład: można też zapłacić gotówką na lekcji." />
    <ApiFeedback error={mutation.error} />
    <ActionButton type="submit" variant="primary" busy={mutation.isPending}>Zapisz dane</ActionButton>
  </form>;
}
