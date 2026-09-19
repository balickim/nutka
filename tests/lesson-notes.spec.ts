import { expect, test } from "@playwright/test";
import type { Route } from "@playwright/test";

const learner = { id: "learner-id", email: "learner@example.test", name: "Jan Kowalski" };
const assignment = { id: "assignment-1", teacher: "teacher-id", learner: learner.id, teacher_name: "Dominika", learner_name: "Jan Kowalski", active: true };
const policy = {
  version: "v1", currency: "PLN", ad_hoc_price_minor: 8000, package_price_minor: 26000, regular_lesson_price_minor: 5000,
  lesson_duration_minutes: 45, start_grid_minutes: 15, participant_buffer_minutes: 5, learner_booking_minimum_hours: 24, learner_change_cutoff_hours: 24,
  booking_horizon_days: 14, package_token_count: 4, package_validity_days: 60, teacher_cancellation_extension_days: 7, contract_monthly_reschedules: 1,
  contract_free_cancellations: 2, contract_replacement_deadline_days: 30, monthly_payment_due_day: 5, contract_end_month: 6, contract_end_day: 30,
};
const note = (id: string, start: string, body: string) => ({ id, lesson: `lesson-${id}`, assignment: assignment.id, lesson_start_at: start, body, materials: [{ id: "material-1", title: "Gama C-dur" }], updated_at: start });
const json = (route: Route, body: unknown) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(body) });

test("a learner reads the latest note on Start and every note on the lessons screen", async ({ page }) => {
  await page.route("**/api/health", (route) => json(route, {}));
  await page.route("**/api/learners/auth/me", (route) => json(route, { record: learner }));
  await page.route("**/api/learners/business-policy", (route) => json(route, policy));
  await page.route("**/api/learners/calendar", (route) => json(route, { assignments: [assignment], availability_rules: [], availability_exceptions: [], commercial_summaries: [], near_term_lessons: [], payment_summary: [], history_summary: [] }));
  await page.route("**/api/learners/assignments/assignment-1/payment-due", (route) => json(route, { assignment: assignment.id, currency: "PLN", total_minor: 0, items: [], open_credit_minor: 0, next_forecast: null, recent_payments: [], instructions: null }));
  await page.route("**/api/learners/assignments/assignment-1/lesson-notes", (route) => json(route, { items: [note("2", "2030-10-09T15:00:00Z", "<p>Refren <strong>wolniej</strong>, tempo 70.</p>"), note("1", "2030-10-02T15:00:00Z", "<p>Pierwsze akordy.</p>")] }));
  await page.route("**/api/learners/assignments/assignment-1/slots**", (route) => json(route, { slots: [] }));
  await page.route("**/api/learners/assignments/assignment-1/commercial-summary", (route) => json(route, { assignment: assignment.id, active_plan: null, payments: { pending: 0, intentionally_unpaid: 0, overdue: 0, credit_minor: 0, currency: "PLN" } }));
  await page.route("**/api/learners/assignments/assignment-1/history**", (route) => json(route, { items: [], page: 1, per_page: 20, total: 0 }));

  await page.goto("/learners");
  await expect(page.getByRole("heading", { name: "Po ostatniej lekcji" })).toBeVisible();
  await expect(page.getByText("Refren wolniej, tempo 70.")).toBeVisible();
  await expect(page.getByText("Pierwsze akordy.")).toHaveCount(0);
  await page.getByRole("link", { name: "Wszystkie notatki" }).click();
  await expect(page).toHaveURL(/\/learners\/lessons\?a=assignment-1#notatki$/);
  await expect(page.getByRole("heading", { name: "Notatki z lekcji" })).toBeVisible();
  await expect(page.getByText("Pierwsze akordy.")).toBeVisible();
  await expect(page.getByRole("link", { name: "Gama C-dur" }).first()).toBeVisible();
});
