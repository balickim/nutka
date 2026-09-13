import { expect, test } from "@playwright/test";
import type { Page, Route } from "@playwright/test";

const teacher = { id: "teacher-1", email: "teacher@example.test", name: "Ada Teacher", timezone: "Europe/Warsaw" };
const learner = { id: "learner-1", email: "learner@example.test", name: "Leo Learner" };
const assignment = {
  id: "assignment-1",
  teacher: teacher.id,
  learner: learner.id,
  teacher_name: "Ada Lovelace",
  learner_name: "Leo Learner",
  active: true,
  default_duration_minutes: 45,
};

const firstSlot = "2030-01-15T10:00:00Z";
const secondSlot = "2030-01-16T10:00:00Z";
const teacherRescheduledStart = "2030-01-15T10:30:00Z";
const learnerRescheduledStart = "2030-01-17T10:00:00Z";

type Lesson = {
  id: string;
  teacher: string;
  learner: string;
  assignment: string;
  start_at: string;
  end_at: string;
  duration_minutes: number;
  status: "scheduled" | "cancelled";
  cancellation_initiator_role?: "teacher" | "learner";
};
type Rule = { id: string; teacher: string; weekday: number; start_time: string; end_time: string; enabled: boolean };
type Mutation = { method: string; path: string; body: Record<string, unknown>; headers: Record<string, string> };
type Fixture = { rules: Rule[]; lessons: Lesson[]; counters: { teacher: number; learner: number }; mutations: Mutation[] };

const lesson = (id: string, startAt: string): Lesson => ({
  id,
  teacher: teacher.id,
  learner: learner.id,
  assignment: assignment.id,
  start_at: startAt,
  end_at: new Date(Date.parse(startAt) + 45 * 60_000).toISOString(),
  duration_minutes: 45,
  status: "scheduled",
});

async function json(route: Route, data: unknown, status = 200) {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(data) });
}

function calendar(fixture: Fixture) {
  return {
    assignments: [assignment],
    availability_rules: fixture.rules,
    availability_exceptions: [],
    lessons: fixture.lessons,
    cancellation_counters: fixture.counters,
  };
}

async function captureMutation(route: Route, fixture: Fixture, path: string) {
  const bodyText = route.request().postData() || "{}";
  let body: Record<string, unknown>;
  try {
    body = JSON.parse(bodyText) as Record<string, unknown>;
  } catch {
    body = {};
  }
  fixture.mutations.push({ method: route.request().method(), path, body, headers: route.request().headers() });
  return body;
}

async function installSchedulingMocks(page: Page, role: "teacher" | "learner", fixture: Fixture) {
  let authenticated = false;
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    if (path === "/api/health") return json(route, {});
    if (path === "/api/teachers/auth/me") return authenticated ? json(route, { record: teacher }) : json(route, {}, 401);
    if (path === "/api/learners/auth/me") return authenticated ? json(route, { record: learner }) : json(route, {}, 401);
    if (path === "/api/collections/teachers/auth-with-password" || path === "/api/collections/learners/auth-with-password") {
      authenticated = true;
      return json(route, { record: path.includes("teachers") ? teacher : learner, token: "" });
    }
    if (path === "/api/teachers/auth/logout" || path === "/api/learners/auth/logout") {
      authenticated = false;
      return json(route, { status: "ok" });
    }
    if (path === "/api/teachers/calendar") return json(route, calendar(fixture));
    if (path === "/api/learners/calendar") return json(route, calendar(fixture));
    if (path === "/api/teachers/availability/rules" && request.method() === "POST") {
      const body = await captureMutation(route, fixture, path);
      const created: Rule = { id: "rule-1", teacher: teacher.id, weekday: Number(body.weekday), start_time: String(body.start_time), end_time: String(body.end_time), enabled: Boolean(body.enabled) };
      fixture.rules = [created];
      return json(route, created);
    }
    if (path === "/api/learners/assignments/assignment-1/slots") {
      const occupied = new Set(fixture.lessons.filter((item) => item.status === "scheduled").map((item) => item.start_at));
      const slots = [firstSlot, secondSlot, "2030-01-18T10:00:00Z"]
        .filter((startAt) => !occupied.has(startAt))
        .map((startAt) => ({ start_at: startAt, end_at: new Date(Date.parse(startAt) + 45 * 60_000).toISOString(), duration_minutes: 45 }));
      return json(route, { teacher: teacher.id, assignment: assignment.id, slots });
    }
    if (path === "/api/learners/assignments/assignment-1/book" && request.method() === "POST") {
      const body = await captureMutation(route, fixture, path);
      const created = lesson(`lesson-${fixture.lessons.length + 1}`, String(body.start_at));
      fixture.lessons.push(created);
      return json(route, created);
    }
    const reschedule = path.match(/^\/api\/(teachers|learners)\/lessons\/(lesson-[12])\/reschedule$/);
    if (reschedule && request.method() === "PATCH") {
      const body = await captureMutation(route, fixture, path);
      const current = fixture.lessons.find((item) => item.id === reschedule[2]);
      if (!current) return json(route, { code: "not_found", message: "Lekcja nie istnieje." }, 404);
      const duration = reschedule[1] === "teachers" && body.duration_minutes ? Number(body.duration_minutes) : current.duration_minutes;
      if (body.start_at) current.start_at = String(body.start_at);
      current.duration_minutes = duration;
      current.end_at = new Date(Date.parse(current.start_at) + duration * 60_000).toISOString();
      return json(route, current);
    }
    const cancellation = path.match(/^\/api\/(teachers|learners)\/lessons\/(lesson-[12])\/cancel$/);
    if (cancellation && request.method() === "POST") {
      await captureMutation(route, fixture, path);
      const current = fixture.lessons.find((item) => item.id === cancellation[2]);
      if (!current) return json(route, { code: "not_found", message: "Lekcja nie istnieje." }, 404);
      current.status = "cancelled";
      current.cancellation_initiator_role = cancellation[1] === "teachers" ? "teacher" : "learner";
      fixture.counters[current.cancellation_initiator_role] += 1;
      return json(route, current);
    }
    if (role === "teacher" && path.startsWith("/api/teachers/")) return json(route, {});
    if (role === "learner" && path.startsWith("/api/learners/")) return json(route, {});
    return route.continue();
  });
}

async function localizedInstant(page: Page, instant: string) {
  return page.evaluate((value) => new Intl.DateTimeFormat("pl-PL", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value)), instant);
}

async function browserDateTimeInput(page: Page, instant: string) {
  return page.evaluate((value) => {
    const date = new Date(value);
    const pad = (part: number) => String(part).padStart(2, "0");
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }, instant);
}

test("teacher and learner complete an isolated scheduling lifecycle", async ({ browser }) => {
  const fixture: Fixture = { rules: [], lessons: [], counters: { teacher: 0, learner: 0 }, mutations: [] };
  const context = await browser.newContext();
  const teacherPage = await context.newPage();
  const learnerPage = await context.newPage();
  await installSchedulingMocks(teacherPage, "teacher", fixture);
  await installSchedulingMocks(learnerPage, "learner", fixture);

  await teacherPage.goto("/teachers/login");
  await teacherPage.getByLabel("Adres e-mail").fill(teacher.email);
  await teacherPage.getByLabel("Hasło").fill("teacher-password");
  await teacherPage.getByRole("button", { name: "Zaloguj się" }).click();
  await expect(teacherPage).toHaveURL(/\/teachers$/);

  await teacherPage.goto("/teachers/availability");
  await teacherPage.getByRole("button", { name: "Dodaj regułę" }).click();
  await expect(teacherPage.locator("article.rule-row strong").filter({ hasText: "Poniedziałek" })).toBeVisible();
  await expect(teacherPage.getByText("16:00–20:00", { exact: true })).toBeVisible();
  await expect(teacherPage.getByText("Reguły: Europe/Warsaw", { exact: true })).toBeVisible();

  await learnerPage.goto("/learners/login");
  await learnerPage.getByLabel("Adres e-mail").fill(learner.email);
  await learnerPage.getByLabel("Hasło").fill("learner-password");
  await learnerPage.getByRole("button", { name: "Zaloguj się" }).click();
  await expect(learnerPage).toHaveURL(/\/learners\/calendar$/);
  await expect(learnerPage.getByRole("heading", { name: "Ada Lovelace", exact: true })).toBeVisible();
  const firstSlotText = await localizedInstant(learnerPage, firstSlot);
  await expect(learnerPage.locator("button.slot-button").first()).toContainText(firstSlotText);

  await learnerPage.locator("button.slot-button").first().click();
  await expect(learnerPage.locator("article.lesson-card")).toHaveCount(1);
  await learnerPage.locator("button.slot-button").first().click();
  await expect(learnerPage.locator("article.lesson-card")).toHaveCount(2);

  await teacherPage.goto("/teachers");
  await expect(teacherPage.locator("article.lesson-card")).toHaveCount(2);
  const firstTeacherLesson = teacherPage.locator("article.lesson-card").first();
  await firstTeacherLesson.getByRole("button", { name: "Przełóż" }).click();
  await teacherPage.locator("#start-lesson-1").fill(await browserDateTimeInput(teacherPage, teacherRescheduledStart));
  await teacherPage.locator("#duration-lesson-1").fill("60");
  await firstTeacherLesson.getByRole("button", { name: "Zapisz termin" }).click();
  await expect(teacherPage.locator("article.lesson-card").first()).toContainText("60 min");
  await expect(teacherPage.locator("article.lesson-card").first()).toContainText(await localizedInstant(teacherPage, teacherRescheduledStart));

  await teacherPage.locator("article.lesson-card").first().getByRole("button", { name: "Odwołaj" }).click();
  await expect(teacherPage.locator("article.lesson-card").first()).toContainText("Odwołana");
  await expect(teacherPage.locator("article.lesson-card").first()).toContainText("Odwołana przez Ciebie");
  await expect(teacherPage.locator(".counter strong").first()).toHaveText("1");

  await learnerPage.reload();
  await expect(learnerPage.locator("article.lesson-card")).toHaveCount(2);
  await expect(learnerPage.locator("article.lesson-card").first()).toContainText("Odwołana przez drugą osobę");
  const scheduledLearnerLesson = learnerPage.locator("article.lesson-card").filter({ hasText: "Zaplanowana" }).first();
  await scheduledLearnerLesson.getByRole("button", { name: "Przełóż" }).click();
  await expect(learnerPage.locator("#duration-lesson-2")).toHaveCount(0);
  await learnerPage.locator("#start-lesson-2").fill(await browserDateTimeInput(learnerPage, learnerRescheduledStart));
  await scheduledLearnerLesson.getByRole("button", { name: "Zapisz termin" }).click();
  await expect(learnerPage.locator("article.lesson-card").filter({ hasText: "Zaplanowana" }).first()).toContainText(await localizedInstant(learnerPage, learnerRescheduledStart));
  await learnerPage.locator("article.lesson-card").filter({ hasText: "Zaplanowana" }).first().getByRole("button", { name: "Odwołaj" }).click();
  await expect(learnerPage.locator("article.lesson-card").nth(1)).toContainText("Odwołana przez Ciebie");
  await expect(learnerPage.locator(".counter strong").first()).toHaveText("1");

  const schedulingMutations = fixture.mutations;
  expect(schedulingMutations.length).toBe(7);
  for (const mutation of schedulingMutations) {
    expect(mutation.headers["x-requested-with"]).toBe("fetch");
    for (const identityField of ["id", "teacher", "learner", "assignment"]) expect(mutation.body).not.toHaveProperty(identityField);
  }
  const learnerBooking = schedulingMutations.find((mutation) => mutation.path.endsWith("/book"));
  const learnerReschedule = schedulingMutations.find((mutation) => mutation.path.includes("/learners/lessons/") && mutation.path.endsWith("/reschedule"));
  expect(learnerBooking?.body).toEqual({ start_at: firstSlot });
  expect(Object.keys(learnerReschedule?.body ?? {})).toEqual(["start_at"]);
  expect(Date.parse(String(learnerReschedule?.body.start_at))).toBe(Date.parse(learnerRescheduledStart));

  await teacherPage.getByRole("button", { name: "Wyloguj się" }).click();
  await expect(teacherPage).toHaveURL(/\/teachers\/login$/);
  await expect(learnerPage).toHaveURL(/\/learners\/calendar$/);
  await expect(learnerPage.getByRole("heading", { name: /Cześć, Leo Learner/ })).toBeVisible();
  await context.close();
});
