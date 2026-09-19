import { expect, test } from "@playwright/test";
import type { Page, Route } from "@playwright/test";

const learner = { id: "learner-id", email: "learner@example.test", name: "Jan Kowalski" };
const assignment = { id: "assignment-1", teacher: "teacher-id", learner: learner.id, teacher_name: "Dominika", learner_name: "Jan Kowalski", active: true };
const policy = {
  version: "v1", currency: "PLN", ad_hoc_price_minor: 8000, package_price_minor: 26000, regular_lesson_price_minor: 5000,
  lesson_duration_minutes: 45, start_grid_minutes: 15, participant_buffer_minutes: 5, learner_booking_minimum_hours: 24, learner_change_cutoff_hours: 24,
  booking_horizon_days: 14, package_token_count: 4, package_validity_days: 60, teacher_cancellation_extension_days: 7, contract_monthly_reschedules: 1,
  contract_free_cancellations: 2, contract_replacement_deadline_days: 30, monthly_payment_due_day: 5, contract_end_month: 6, contract_end_day: 30,
};
const paymentDue = {
  assignment: assignment.id, currency: "PLN", total_minor: 20000, open_credit_minor: 0,
  items: [{ charge: "charge-1", kind: "contract_month", period: "2030-10", amount_minor: 20000, due_on: "2030-10-05", overdue: true }],
  next_forecast: { month: "2030-11", lesson_count: 4, amount_minor: 20000, due_on: "2030-11-05" },
  recent_payments: [],
  instructions: { account_holder: "Dominika Nowak", iban: "PL61109010140000071219812874", note: "" },
};

const json = (route: Route, body: unknown) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(body) });

async function mockLearner(page: Page) {
  await page.route("**/api/health", (route) => json(route, {}));
  await page.route("**/api/learners/auth/me", (route) => json(route, { record: learner }));
  await page.route("**/api/learners/business-policy", (route) => json(route, policy));
  await page.route("**/api/learners/calendar", (route) => json(route, { assignments: [assignment], availability_rules: [], availability_exceptions: [], commercial_summaries: [], near_term_lessons: [], payment_summary: [], history_summary: [] }));
  await page.route("**/api/learners/assignments/assignment-1/payment-due", (route) => json(route, paymentDue));
}

test("a learner with an overdue charge finds the amount, account, title, and QR code", async ({ page }) => {
  await mockLearner(page);
  await page.goto("/learners");
  await expect(page.getByRole("heading", { name: "Do zapłaty: 200,00 zł" })).toBeVisible();
  await expect(page.getByText("Po terminie")).toBeVisible();
  await page.getByRole("link", { name: "Jak zapłacić" }).click();
  await expect(page).toHaveURL(/\/learners\/payments\?a=assignment-1$/);
  await expect(page.getByText("Lekcje, październik 2030")).toBeVisible();
  await expect(page.getByText("61 1090 1014 0000 0712 1981 2874")).toBeVisible();
  await expect(page.getByText("Lekcje Jan Kowalski 10.2030")).toBeVisible();
  await expect(page.getByRole("img", { name: /Kod QR przelewu/ })).toBeVisible();
  await expect(page.getByText(/Listopad 2030: 4 lekcje, 200,00 zł\. Termin płatności: wtorek, 5 listopada 2030/)).toBeVisible();
});

test("a learner with nothing due sees no payment card and no QR code", async ({ page }) => {
  await mockLearner(page);
  await page.route("**/api/learners/assignments/assignment-1/payment-due", (route) => json(route, { ...paymentDue, total_minor: 0, items: [] }));
  await page.goto("/learners");
  await expect(page.getByRole("heading", { name: "Najbliższa lekcja" })).toBeVisible();
  await expect(page.getByRole("heading", { name: /Do zapłaty/ })).toHaveCount(0);
  await page.goto("/learners/payments");
  await expect(page.getByText("Nie masz teraz nic do zapłaty.")).toBeVisible();
  await expect(page.getByRole("img", { name: /Kod QR przelewu/ })).toHaveCount(0);
});
