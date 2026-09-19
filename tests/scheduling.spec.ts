import { expect, test } from "@playwright/test";
import type { Page, Route } from "@playwright/test";

const teacher = { id: "teacher-1", email: "teacher@example.test", name: "Ada Teacher", timezone: "Europe/Warsaw" };
const learner = { id: "learner-1", email: "learner@example.test", name: "Leo Learner" };
const assignment = { id: "assignment-1", teacher: teacher.id, learner: learner.id, teacher_name: "Ada Teacher", learner_name: "Leo Learner", active: true };
const starts = ["2030-09-16T15:00:00Z", "2030-09-17T15:00:00Z", "2030-09-18T15:00:00Z", "2030-09-19T15:00:00Z", "2030-09-20T15:00:00Z"];
const apiRequestPattern = /^https?:\/\/[^/]+\/api\//;
const policy = {
  version: "2026-09-lesson-plans-v1", currency: "PLN", ad_hoc_price_minor: 8000, package_price_minor: 26000, regular_lesson_price_minor: 5000,
  lesson_duration_minutes: 45, start_grid_minutes: 15, participant_buffer_minutes: 5, learner_booking_minimum_hours: 24, learner_change_cutoff_hours: 24,
  booking_horizon_days: 14, package_token_count: 4, package_validity_days: 60, teacher_cancellation_extension_days: 7, contract_monthly_reschedules: 1,
  contract_free_cancellations: 2, contract_replacement_deadline_days: 30, monthly_payment_due_day: 5, contract_end_month: 6, contract_end_day: 30,
};

type Mutation = { path: string; body: Record<string, unknown>; headers: Record<string, string> };

function lesson(id: string, startAt: string, planType: "ad_hoc" | "package" | "regular_contract" = "ad_hoc") {
  const endAt = new Date(Date.parse(startAt) + 45 * 60_000).toISOString();
  return {
    id,
    teacher: teacher.id,
    learner: learner.id,
    assignment: assignment.id,
    start_at: startAt,
    end_at: endAt,
    duration_minutes: 45,
    plan_type: planType,
    package_token: planType === "package" ? `token-${id}` : null,
    contract: planType === "regular_contract" ? "contract-1" : null,
    policy_version: "2026-09-lesson-plans-v1",
    unit_price_minor: planType === "ad_hoc" ? 8000 : planType === "regular_contract" ? 5000 : 0,
    currency: "PLN",
    schedule_state: "scheduled" as const,
    outcome: null,
    protected_interval: { start_at: new Date(Date.parse(startAt) - 5 * 60_000).toISOString(), end_at: new Date(Date.parse(endAt) + 5 * 60_000).toISOString() },
  };
}

function calendar(nearTerm: ReturnType<typeof lesson>[] = [], later: ReturnType<typeof lesson>[] = []) {
  return {
    assignments: [assignment],
    availability_rules: [],
    availability_exceptions: [],
    commercial_summaries: [],
    near_term_lessons: nearTerm,
    later_contract_lessons: later,
    unresolved_work: { awaiting_outcome: 0, pending_settlement: 0, unpaid_charges: 0, overdue_charges: 0 },
  };
}

const emptyHistory = { items: [], page: 1, per_page: 20, total: 0 };
const emptyFinancialWork = { charges: [], entries: [], credits: [] };
const emptyUnresolved = { awaiting_outcome: [], pending_settlements: [], unpaid_charges: [], overdue_charges: [] };

async function json(route: Route, value: unknown, status = 200) {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(value) });
}

async function bodyOf(route: Route): Promise<Record<string, unknown>> {
  return JSON.parse(route.request().postData() || "{}") as Record<string, unknown>;
}

async function installSession(page: Page, role: "teacher" | "learner") {
  await page.route("**/api/health", (route) => json(route, {}));
  await page.route(`**/api/${role === "teacher" ? "teachers" : "learners"}/auth/me`, (route) => json(route, { record: role === "teacher" ? teacher : learner }));
}

function packageValue(reserved: number) {
  return {
    id: "package-1",
    assignment: assignment.id,
    status: "open",
    purchased_on: "2030-09-01",
    valid_through: "2030-10-30",
    price_minor: 26000,
    currency: "PLN",
    policy_version: "2026-09-lesson-plans-v1",
    tokens: [0, 1, 2, 3].map((index) => ({ id: `token-${index + 1}`, ordinal: index + 1, state: index < reserved ? "reserved" : "available", lesson: index < reserved ? `lesson-${index + 1}` : null })),
  };
}

function packageSummary(reserved: number) {
  return {
    assignment: assignment.id,
    active_plan: reserved < 4 ? "package" : "package",
    package: { id: "package-1", valid_through: "2030-10-30", token_balance: { available: 4 - reserved, reserved, used: 0, expired: 0, invalidated: 0 } },
    contract: null,
    payments: { pending: 0, intentionally_unpaid: 0, overdue: 0, credit_minor: 0, currency: "PLN" },
  };
}

test("teacher purchase, four package bookings, precedence, and ad hoc settlement stay explicit", async ({ browser }) => {
  const context = await browser.newContext();
  const teacherPage = await context.newPage();
  const learnerPage = await context.newPage();
  const mutations: Mutation[] = [];
  let purchased = false;
  let reserved = 0;
  const lessons: ReturnType<typeof lesson>[] = [];
  await installSession(teacherPage, "teacher");
  await teacherPage.route(apiRequestPattern, async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/health") return json(route, {});
    if (path === "/api/teachers/auth/me") return json(route, { record: teacher });
    if (path === "/api/teachers/business-policy") return json(route, policy);
    if (path === "/api/teachers/calendar") return json(route, calendar(lessons));
    if (path === "/api/teachers/unresolved-work") return json(route, { ...emptyUnresolved, pending_settlements: [{ lesson: "settlement-1", assignment: assignment.id, amount_minor: 8000, currency: "PLN" }] });
    if (path === "/api/teachers/financial-work") return json(route, emptyFinancialWork);
    if (path.endsWith("/commercial-summary")) return json(route, purchased ? packageSummary(reserved) : { ...packageSummary(4), active_plan: "ad_hoc", package: null });
    if (path.endsWith("/packages") && route.request().method() === "GET") return json(route, { packages: purchased ? [packageValue(reserved)] : [] });
    if (path.endsWith("/contracts")) return json(route, { contracts: [] });
    if (path.endsWith("/history")) return json(route, emptyHistory);
    if (path === "/api/teachers/assignments/assignment-1/packages") {
      const body = await bodyOf(route);
      mutations.push({ path, body, headers: route.request().headers() });
      purchased = true;
      return json(route, packageValue(reserved), 201);
    }
    if (path === "/api/teachers/lessons/settlement-1/settlement") {
      const body = await bodyOf(route);
      mutations.push({ path, body, headers: route.request().headers() });
      return json(route, { lesson: "settlement-1", assignment: assignment.id, amount_minor: 8000, currency: "PLN", settlement_state: body.settlement });
    }
    return json(route, { code: "not_found", message: "Not found." }, 404);
  });

  await teacherPage.goto("/teachers/students/assignment-1");
  await teacherPage.getByRole("tab", { name: "Plan" }).click();
  await teacherPage.getByRole("button", { name: "Oznacz zakup pakietu" }).click();
  await teacherPage.getByRole("button", { name: "Dalej" }).click();
  await teacherPage.getByRole("button", { name: "Zapisz zakup" }).click();
  await expect(teacherPage.getByText("Dostępny", { exact: false }).first()).toBeVisible();
  await teacherPage.goto("/teachers/billing");
  await teacherPage.getByRole("button", { name: "Zapisz rozliczenie" }).click();

  await installSession(learnerPage, "learner");
  await learnerPage.route(apiRequestPattern, async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/health") return json(route, {});
    if (path === "/api/learners/auth/me") return json(route, { record: learner });
    if (path === "/api/learners/business-policy") return json(route, policy);
    if (path === "/api/learners/calendar") return json(route, calendar(lessons));
    if (path.endsWith("/commercial-summary")) return json(route, packageSummary(reserved));
    if (path.endsWith("/history")) return json(route, emptyHistory);
    if (path.endsWith("/slots")) {
      const occupied = new Set(lessons.map((item) => item.start_at));
      const slots = starts.filter((start) => !occupied.has(start)).map((start) => ({ start_at: start, end_at: new Date(Date.parse(start) + 45 * 60_000).toISOString(), duration_minutes: 45, protected_interval: { start_at: start, end_at: new Date(Date.parse(start) + 50 * 60_000).toISOString() } }));
      return json(route, { teacher: teacher.id, assignment: assignment.id, policy_version: "2026-09-lesson-plans-v1", slots });
    }
    if (path.endsWith("/book")) {
      const body = await bodyOf(route);
      mutations.push({ path, body, headers: route.request().headers() });
      const plan = reserved < 4 ? "package" : "ad_hoc";
      const created = lesson(`lesson-${lessons.length + 1}`, String(body.start_at), plan);
      lessons.push(created);
      if (plan === "package") reserved++;
      return json(route, created, 201);
    }
    return json(route, { code: "not_found", message: "Not found." }, 404);
  });

  await learnerPage.goto("/learners/lessons");
  await learnerPage.locator("summary", { hasText: "Zarezerwuj lekcję" }).click();
  for (let expected = 1; expected <= 5; expected++) {
    await learnerPage.locator("button.slot-button").first().click();
    await learnerPage.getByRole("dialog").getByRole("button", { name: "Zarezerwuj lekcję" }).click();
    await expect(learnerPage.locator("article.lesson-card")).toHaveCount(expected);
  }
  await expect(learnerPage.locator("article.lesson-card").filter({ hasText: "Pakiet lekcji" })).toHaveCount(4);
  await expect(learnerPage.locator("article.lesson-card").filter({ hasText: "Pojedyncza lekcja" })).toHaveCount(1);
  const bookingMutations = mutations.filter((item) => item.path.endsWith("/book"));
  expect(bookingMutations).toHaveLength(5);
  expect(bookingMutations.slice(0, 4).every((item) => !Object.hasOwn(item.body, "plan_type"))).toBe(true);
  expect(mutations.find((item) => item.path.endsWith("/settlement"))?.body).toEqual({ settlement: "paid" });
  expect(mutations.every((item) => item.headers["x-requested-with"] === "fetch")).toBe(true);
  await context.close();
});

const contract = {
  id: "contract-1",
  assignment: assignment.id,
  status: "active",
  start_on: "2030-09-16",
  end_on: "2031-06-30",
  weekday: 1,
  start_time: "17:00",
  price_minor: 5000,
  currency: "PLN",
  policy_version: "2026-09-lesson-plans-v1",
  remaining_monthly_reschedules: 1,
  remaining_free_cancellations: 2,
};

function contractSummary(endOn = contract.end_on) {
  return {
    assignment: assignment.id,
    active_plan: "regular_contract",
    package: null,
    contract: { id: contract.id, status: contract.status, start_on: contract.start_on, end_on: endOn, remaining_monthly_reschedules: 1, remaining_free_cancellations: 2, price_minor: 5000, currency: "PLN" },
    payments: { pending: 1, intentionally_unpaid: 0, overdue: 0, credit_minor: 0, currency: "PLN" },
  };
}

test("regular contract exposes series, learner horizon controls, forecast, notice, and payment", async ({ browser }) => {
  const context = await browser.newContext();
  const teacherPage = await context.newPage();
  const learnerPage = await context.newPage();
  const mutations: Mutation[] = [];
  let active = false;
  let effectiveEnd = contract.end_on;
  let occurrence = lesson("contract-lesson-1", starts[0], "regular_contract");
  const later = lesson("contract-lesson-later", "2031-03-03T16:00:00Z", "regular_contract");
  const charge = { id: "charge-1", assignment: assignment.id, source_type: "regular_contract", source_id: contract.id, period: "2030-09", original_amount_minor: 15000, current_amount_minor: 15000, currency: "PLN", settlement_state: "pending", derived_state: "pending", overdue: false, due_on: "2030-09-05" };
  await installSession(teacherPage, "teacher");
  await teacherPage.route(apiRequestPattern, async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/health") return json(route, {});
    if (path === "/api/teachers/auth/me") return json(route, { record: teacher });
    if (path === "/api/teachers/business-policy") return json(route, policy);
    if (path === "/api/teachers/calendar") return json(route, calendar(active ? [occurrence] : [], active ? [later] : []));
    if (path === "/api/teachers/unresolved-work") return json(route, { ...emptyUnresolved, unpaid_charges: active ? [charge] : [] });
    if (path === "/api/teachers/financial-work") return json(route, { ...emptyFinancialWork, charges: active ? [charge] : [] });
    if (path.endsWith("/commercial-summary")) return json(route, active ? contractSummary(effectiveEnd) : { ...contractSummary(), active_plan: "ad_hoc", contract: null });
    if (path.endsWith("/packages")) return json(route, { packages: [] });
    if (path === "/api/teachers/assignments/assignment-1/contracts" && route.request().method() === "GET") return json(route, { contracts: active ? [{ ...contract, effective_end_on: effectiveEnd }] : [] });
    if (path.endsWith("/history")) return json(route, emptyHistory);
    if (path === "/api/teachers/assignments/assignment-1/contracts" && route.request().method() === "POST") {
      const body = await bodyOf(route);
      mutations.push({ path, body, headers: route.request().headers() });
      active = true;
      return json(route, contract, 201);
    }
    if (path === "/api/teachers/contracts/contract-1/series") return json(route, { contract, near_term: [occurrence], later: [later] });
    if (path === "/api/teachers/contracts/contract-1/months") return json(route, { months: [{ id: "month-1", contract: contract.id, assignment: assignment.id, month: "2030-09", billable_count: 3, forecast_amount_minor: 15000, currency: "PLN", forecast: true }] });
    if (path === "/api/teachers/charges/charge-1/payment") {
      const body = await bodyOf(route);
      mutations.push({ path, body, headers: route.request().headers() });
      return json(route, { ...charge, settlement_state: body.settlement, derived_state: body.settlement });
    }
    return json(route, { code: "not_found", message: "Not found." }, 404);
  });

  await teacherPage.goto("/teachers/students/assignment-1");
  await teacherPage.getByRole("tab", { name: "Plan" }).click();
  await teacherPage.getByRole("button", { name: "Aktywuj umowę" }).click();
  await teacherPage.getByLabel("Data początku").fill("2030-09-16");
  await teacherPage.getByRole("button", { name: "Dalej" }).click();
  await expect(teacherPage.getByRole("dialog")).toContainText("50,00 zł za lekcję");
  await teacherPage.getByRole("dialog").getByRole("button", { name: "Aktywuj umowę" }).click();
  await teacherPage.getByText("Seria lekcji").click();
  await expect(teacherPage.getByText("Najbliższe", { exact: true })).toBeVisible();
  await teacherPage.getByText("Prognozy i należności miesięczne").click();
  await expect(teacherPage.getByText(/2030-09.*150,00 zł.*prognoza/)).toBeVisible();
  await teacherPage.goto("/teachers/billing");
  await teacherPage.getByRole("button", { name: "Zapisz płatność" }).click();

  await installSession(learnerPage, "learner");
  await learnerPage.route(apiRequestPattern, async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/health") return json(route, {});
    if (path === "/api/learners/auth/me") return json(route, { record: learner });
    if (path === "/api/learners/business-policy") return json(route, policy);
    if (path === "/api/learners/calendar") return json(route, { ...calendar([occurrence]), payment_summary: [contractSummary(effectiveEnd).payments], history_summary: [{ assignment: assignment.id, event_count: 2 }] });
    if (path.endsWith("/commercial-summary")) return json(route, contractSummary(effectiveEnd));
    if (path.endsWith("/history")) return json(route, { items: [{ id: "event-1", assignment: assignment.id, aggregate_type: "contract", aggregate_id: contract.id, event_type: "contract_activated", actor_role: "teacher", event_at: "2030-09-01T10:00:00Z" }], page: 1, per_page: 20, total: 1 });
    if (path.endsWith("/slots")) return json(route, { teacher: teacher.id, assignment: assignment.id, policy_version: contract.policy_version, slots: [] });
    if (path.endsWith("/reschedule")) {
      const body = await bodyOf(route);
      mutations.push({ path, body, headers: route.request().headers() });
      occurrence = lesson(occurrence.id, String(body.start_at), "regular_contract");
      return json(route, occurrence);
    }
    if (path.endsWith("/cancel")) {
      const body = await bodyOf(route);
      mutations.push({ path, body, headers: route.request().headers() });
      occurrence = { ...occurrence, schedule_state: "cancelled" };
      return json(route, occurrence);
    }
    if (path === "/api/learners/contracts/contract-1/notice") {
      const body = await bodyOf(route);
      mutations.push({ path, body, headers: route.request().headers() });
      effectiveEnd = "2030-10-31";
      return json(route, { ...contract, status: "notice_given", effective_end_on: effectiveEnd });
    }
    return json(route, { code: "not_found", message: "Not found." }, 404);
  });

  await learnerPage.goto("/learners/lessons");
  await expect(learnerPage.getByText("Stałe terminy wynikają z umowy")).toBeVisible();
  await expect(learnerPage.locator("button.slot-button")).toHaveCount(0);
  await learnerPage.getByRole("button", { name: "Przełóż" }).click();
  await learnerPage.getByLabel("Nowy termin").fill("2030-09-23T17:00");
  await learnerPage.getByRole("button", { name: "Zapisz termin" }).click();
  await learnerPage.getByRole("dialog").getByRole("button", { name: "Przełóż lekcję" }).click();
  await learnerPage.getByRole("button", { name: "Odwołaj" }).click();
  await learnerPage.getByRole("dialog").getByRole("button", { name: "Odwołaj lekcję" }).click();
  await learnerPage.getByRole("button", { name: "Złóż wypowiedzenie" }).click();
  await learnerPage.getByRole("dialog").getByRole("button", { name: "Złóż wypowiedzenie" }).click();
  await expect(learnerPage.getByText(/Umowa od poniedziałek, 16 września 2030 do czwartek, 31 października 2030/)).toBeVisible();
  expect(mutations.find((item) => item.path.endsWith("/reschedule"))?.body).toEqual({ start_at: "2030-09-23T15:00:00.000Z" });
  expect(mutations.find((item) => item.path.endsWith("/notice"))?.body).toEqual({});
  expect(mutations.find((item) => item.path.endsWith("/payment"))?.body).toEqual({ settlement: "paid" });
  await context.close();
});

test("availability requires explicit near-term resolution and reviews distant omission before one save", async ({ page }) => {
  const near = lesson("near-lesson", starts[0], "ad_hoc");
  const later = lesson("later-contract", "2031-03-03T16:00:00Z", "regular_contract");
  let committed = false;
  let commitBody: Record<string, unknown> | undefined;
  await installSession(page, "teacher");
  await page.route(apiRequestPattern, async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/health") return json(route, {});
    if (path === "/api/teachers/auth/me") return json(route, { record: teacher });
    if (path === "/api/teachers/business-policy") return json(route, { ...policy, booking_horizon_days: 12 });
    if (path === "/api/teachers/calendar") return json(route, calendar(committed ? [{ ...near, schedule_state: "cancelled" }] : [near], [later]));
    if (path === "/api/teachers/availability/preview") return json(route, { proposal: await bodyOf(route), near_term_conflicts: [{ lesson: near.id, plan: "ad_hoc", start_at: near.start_at, allowed_resolutions: ["cancel", "reschedule"] }], distant_effects: [{ occurrence: later.id, effect: "omit", start_at: later.start_at }], preview_version: "opaque-v1" });
    if (path === "/api/teachers/availability/commit") {
      commitBody = await bodyOf(route);
      committed = true;
      return json(route, { preview: { proposal: commitBody.proposal, near_term_conflicts: [], distant_effects: [], preview_version: "opaque-v1" }, availability_rule: { id: "rule-1", teacher: teacher.id, weekday: 1, start_time: "16:00", end_time: "20:00", enabled: true } });
    }
    return json(route, { code: "not_found", message: "Not found." }, 404);
  });

  await page.goto("/teachers/availability");
  await expect(page.getByRole("heading", { name: "Lekcje w 12 dniach" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Dalsze stałe rezerwacje" })).toBeVisible();
  await page.getByRole("button", { name: "Sprawdź regułę" }).click();
  const save = page.getByRole("dialog").getByRole("button", { name: "Zapisz zmiany" });
  await expect(save).toBeDisabled();
  await expect(page.getByRole("dialog")).toContainText("lekcja zostanie pominięta");
  await page.getByLabel(/Co zrobić z lekcją/).selectOption("cancel");
  await expect(save).toBeEnabled();
  await save.click();
  expect(commitBody).toMatchObject({ preview_version: "opaque-v1", resolutions: [{ lesson: "near-lesson", action: "cancel" }] });
  await expect(page.locator("article.lesson-card").filter({ hasText: "Odwołana" })).toHaveCount(1);
});
