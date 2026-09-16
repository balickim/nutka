import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";

async function mockUnauthenticated(page: Page) {
  await page.route("**/api/learners/auth/me", (route) => route.fulfill({ status: 401, body: "{}" }));
  await page.route("**/api/health", (route) => route.fulfill({ status: 200, body: "{}" }));
}

async function mockEmptyLearnerCalendar(page: Page) {
  await page.route("**/api/learners/calendar", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ assignments: [], availability_rules: [], availability_exceptions: [], commercial_summaries: [], near_term_lessons: [], payment_summary: [], history_summary: [] }),
    }),
  );
}

const learner = {
  id: "learner-id",
  email: "learner@example.test",
  name: "Test Learner",
};

const policy = {
  version: "v1", currency: "PLN", ad_hoc_price_minor: 8000, package_price_minor: 26000, regular_lesson_price_minor: 5000,
  lesson_duration_minutes: 45, start_grid_minutes: 15, participant_buffer_minutes: 5, learner_booking_minimum_hours: 24, learner_change_cutoff_hours: 24,
  booking_horizon_days: 14, package_token_count: 4, package_validity_days: 60, teacher_cancellation_extension_days: 7, contract_monthly_reschedules: 1,
  contract_free_cancellations: 2, contract_replacement_deadline_days: 30, monthly_payment_due_day: 5, contract_end_month: 6, contract_end_day: 30,
};

async function mockLearnerPolicy(page: Page) {
  await page.route("**/api/learners/business-policy", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(policy) }));
}

test("unauthenticated learners are sent to login", async ({ page }) => {
  await mockUnauthenticated(page);
  await page.goto("/learners/calendar");
  await expect(page).toHaveURL(/\/learners\/login/);
  await expect(page.getByRole("heading", { name: "Zaloguj się", exact: true })).toBeVisible();
});

test("invalid credentials stay on login with a generic error", async ({ page }) => {
  await mockUnauthenticated(page);
  await page.route("**/api/collections/learners/auth-with-password", (route) =>
    route.fulfill({ status: 400, body: JSON.stringify({ message: "Failed to authenticate." }) }),
  );
  await page.goto("/learners/login");
  await page.getByLabel("Adres e-mail").fill("learner@example.test");
  await page.getByLabel("Hasło").fill("wrong-password");
  await page.getByRole("button", { name: "Zaloguj się" }).click();
  await expect(page).toHaveURL(/\/learners\/login/);
  await expect(page.getByRole("alert")).toContainText("Sprawdź e-mail i hasło");
  expect(await page.evaluate(() => localStorage.length)).toBe(0);
});

test("successful login reaches the learner home and logout returns to login", async ({ page }) => {
  let authenticated = false;
  await page.route("**/api/learners/auth/me", (route) =>
    route.fulfill(authenticated ? { status: 200, body: JSON.stringify({ record: learner }) } : { status: 401, body: "{}" }),
  );
  await mockEmptyLearnerCalendar(page);
  await mockLearnerPolicy(page);
  await page.route("**/api/health", (route) => route.fulfill({ status: 200, body: "{}" }));
  await page.route("**/api/collections/learners/auth-with-password", async (route) => {
    authenticated = true;
    await route.fulfill({ status: 200, body: JSON.stringify({ record: learner, token: "" }) });
  });
  await page.route("**/api/learners/auth/logout", (route) => route.fulfill({ status: 200, body: JSON.stringify({ status: "ok" }) }));

  await page.goto("/learners/login");
  await page.getByLabel("Adres e-mail").fill(learner.email);
  await page.getByLabel("Hasło").fill("local-password");
  await page.getByRole("button", { name: "Zaloguj się" }).click();
  await expect(page).toHaveURL(/\/learners\/calendar$/);
  await expect(page.getByRole("heading", { name: /Cześć, Test Learner/ })).toBeVisible();
  expect(await page.evaluate(() => ({ localStorage: localStorage.length, cookie: document.cookie }))).toEqual({ localStorage: 0, cookie: "" });

  await page.getByRole("button", { name: "Wyloguj się" }).click();
  await expect(page).toHaveURL(/\/learners\/login/);
});

test("an expired server session returns to login", async ({ page }) => {
  await page.route("**/api/learners/auth/me", (route) =>
    route.fulfill({
      status: 200,
      body: JSON.stringify({ record: learner, session_expires_at: new Date(Date.now() + 300) }),
    }),
  );
  await mockEmptyLearnerCalendar(page);
  await mockLearnerPolicy(page);
  await page.route("**/api/health", (route) => route.fulfill({ status: 200, body: "{}" }));
  await page.goto("/learners/calendar");
  await expect(page.getByRole("heading", { name: /Cześć, Test Learner/ })).toBeVisible();
  await expect(page).toHaveURL(/\/learners\/login/, { timeout: 5000 });
});

test("logout synchronizes across learner tabs without sharing session data", async ({ browser }) => {
  const context = await browser.newContext();
  const first = await context.newPage();
  const second = await context.newPage();
  await first.route("**/api/learners/auth/me", (route) => route.fulfill({ status: 401, body: "{}" }));
  await second.route("**/api/learners/auth/me", (route) => route.fulfill({ status: 200, body: JSON.stringify({ record: learner }) }));
  await mockEmptyLearnerCalendar(first);
  await mockEmptyLearnerCalendar(second);
  await mockLearnerPolicy(first);
  await mockLearnerPolicy(second);
  for (const page of [first, second]) {
    await page.route("**/api/health", (route) => route.fulfill({ status: 200, body: "{}" }));
  }
  await first.route("**/api/collections/learners/auth-with-password", (route) => route.fulfill({ status: 200, body: JSON.stringify({ record: learner, token: "" }) }));
  await first.route("**/api/learners/auth/logout", (route) => route.fulfill({ status: 200, body: JSON.stringify({ status: "ok" }) }));

  await first.goto("/learners/login");
  await first.getByLabel("Adres e-mail").fill(learner.email);
  await first.getByLabel("Hasło").fill("local-password");
  await first.getByRole("button", { name: "Zaloguj się" }).click();
  await expect(first).toHaveURL(/\/learners\/calendar$/);
  await second.goto("/learners/calendar");
  await expect(second.getByRole("heading", { name: /Cześć, Test Learner/ })).toBeVisible();
  await first.getByRole("button", { name: "Wyloguj się" }).click();
  await expect(second).toHaveURL(/\/learners\/login/, { timeout: 5000 });
  await context.close();
});
