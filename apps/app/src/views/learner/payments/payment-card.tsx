// Shows the amount due on the learner Start screen. It renders nothing while the read loads, fails, or finds nothing due.

import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { formatMoney } from "../../../money";
import { paymentDueQuery } from "../../../query/payments";
import { formatLocalDate } from "../../../time/schedule";

export function PaymentCard({ accountId, assignmentId }: { accountId: string; assignmentId: string }) {
  const due = useQuery(paymentDueQuery(accountId, assignmentId));
  if (!due.data || due.data.total_minor <= 0) return null;
  const first = due.data.items.find((item) => item.due_on);
  const overdue = due.data.items.some((item) => item.overdue);
  return <section className="panel-section payment-card">
    <h2>Do zapłaty: {formatMoney(due.data.total_minor, due.data.currency)}</h2>
    {overdue ? <p><span className="status-badge status-overdue">Po terminie</span></p> : first?.due_on ? <p className="supporting-copy">Termin: {formatLocalDate(first.due_on)}.</p> : null}
    <Link className="btn btn-primary btn-sm" to="/learners/payments" search={{ a: assignmentId }}>Jak zapłacić</Link>
  </section>;
}
