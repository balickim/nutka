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
      body: JSON.stringify({ assignments: [], availability_rules: [], availability_exceptions: [], lessons: [], cancellation_counters: { teacher: 0, learner: 0 } }),
    }),
  );
}

const learner = {
  id: "learner-id",
  email: "learner@example.test",
  name: "Test Learner",
};

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
