// Defines the learner payment-due read and the teacher transfer details contract, and requests both through the shared transport.

import { apiRequest } from "./transport";

export type PaymentDetails = { account_holder: string; iban: string; note: string };

export type DueItem = {
  charge: string;
  kind: "contract_month" | "lesson";
  period?: string;
  lesson_start_at?: string;
  amount_minor: number;
  due_on?: string;
  overdue: boolean;
};

export type RecentPayment = Omit<DueItem, "due_on" | "overdue"> & { paid_at: string };

export type PaymentDue = {
  assignment: string;
  currency: "PLN";
  total_minor: number;
  items: DueItem[];
  open_credit_minor: number;
  next_forecast: { month: string; lesson_count: number; amount_minor: number; due_on: string } | null;
  recent_payments: RecentPayment[];
  instructions: PaymentDetails | null;
};

export const fetchPaymentDue = (assignmentId: string, signal?: AbortSignal) =>
  apiRequest<PaymentDue>(`/api/learners/assignments/${encodeURIComponent(assignmentId)}/payment-due`, { signal });

export const fetchPaymentDetails = (signal?: AbortSignal) =>
  apiRequest<PaymentDetails>("/api/teachers/payment-details", { signal });

export const savePaymentDetails = (body: PaymentDetails) =>
  apiRequest<PaymentDetails>("/api/teachers/payment-details", { method: "PUT", body });
