import { expect, test } from "@playwright/test";
import type { Page, Route } from "@playwright/test";

const now = new Date("2030-10-09T14:00:00Z");
const teacher = { id: "teacher-1", email: "teacher@example.test", name: "Dominika", timezone: "Europe/Warsaw" };
const learner = { id: "learner-1", email: "learner@example.test", name: "Jan Kowalski" };
const assignment = { id: "assignment-1", teacher: teacher.id, learner: learner.id, teacher_name: "Dominika", learner_name: "Jan Kowalski", active: true };
const apiRequestPattern = /^https?:\/\/[^/]+\/api\//;
const policy = {
  version: "v1", currency: "PLN", ad_hoc_price_minor: 8000, package_price_minor: 26000, regular_lesson_price_minor: 5000,
  lesson_duration_minutes: 45, start_grid_minutes: 15, participant_buffer_minutes: 5, learner_booking_minimum_hours: 24, learner_change_cutoff_hours: 24,
  booking_horizon_days: 14, package_token_count: 4, package_validity_days: 60, teacher_cancellation_extension_days: 7, contract_monthly_reschedules: 1,
  contract_free_cancellations: 2, contract_replacement_deadline_days: 30, monthly_payment_due_day: 5, contract_end_month: 6, contract_end_day: 30,
};
const lesson = {
  id: "lesson-1", teacher: teacher.id, learner: learner.id, assignment: assignment.id, start_at: "2030-10-09T11:00:00Z", end_at: "2030-10-09T11:45:00Z", duration_minutes: 45,
  plan_type: "ad_hoc", package_token: null, contract: null, policy_version: "v1", unit_price_minor: 8000, currency: "PLN", schedule_state: "scheduled", outcome: null as string | null,
  protected_interval: { start_at: "2030-10-09T10:55:00Z", end_at: "2030-10-09T11:50:00Z" },
};

type Task = { id: string; title: string };
type Session = { practiced_on: string; minutes: number | null; tasks: string[]; comment: string };

async function json(route: Route, value: unknown, status = 200) {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(value) });
}

function summaryOf(sessions: Session[], tasks: Task[]) {
  const days = new Set(sessions.map((session) => session.practiced_on));
  const counts = tasks.map((task) => ({ task: task.id, title: task.title, count: sessions.filter((session) => session.tasks.includes(task.id)).length })).filter((item) => item.count > 0);
  return { since_on: "2030-10-09", days: days.size, minutes: sessions.reduce((sum, session) => sum + (session.minutes ?? 0), 0), sessions: sessions.length, tasks: counts, comments: sessions.filter((session) => session.comment).map((session) => ({ practiced_on: session.practiced_on, comment: session.comment })) };
}

test("the teacher sets tasks after a lesson, the learner records practice, and the teacher sees the summary on Today", async ({ browser }) => {
  const tasks: (Task & Record<string, unknown>)[] = [];
  const sessions: Session[] = [];
  const writes: { path: string; body: Record<string, unknown> }[] = [];
  const context = await browser.newContext();
  const pages: Page[] = [await context.newPage(), await context.newPage()];
  const [teacherPage, learnerPage] = pages;
  for (const page of pages) {
    await page.clock.setFixedTime(now);
    await page.route(apiRequestPattern, async (route) => {
      const request = route.request();
      const url = new URL(request.url());
      const path = url.pathname;
      const persona = path.startsWith("/api/teachers/") ? "teachers" : "learners";
      const body = request.method() === "GET" ? {} : (request.postDataJSON() as Record<string, unknown>);
      if (request.method() !== "GET") writes.push({ path, body });
      if (path === "/api/health") return json(route, {});
      if (path === `/api/${persona}/auth/me`) return json(route, { record: persona === "teachers" ? teacher : learner });
      if (path === `/api/${persona}/business-policy`) return json(route, policy);
      if (path === `/api/${persona}/calendar`) return json(route, { assignments: [assignment], availability_rules: [], availability_exceptions: [], commercial_summaries: [], near_term_lessons: [lesson], later_contract_lessons: [], unresolved_work: { awaiting_outcome: 0, pending_settlement: 0, unpaid_charges: 0, overdue_charges: 0 } });
      if (path === "/api/teachers/unresolved-work") return json(route, { awaiting_outcome: [], pending_settlements: [], unpaid_charges: [], overdue_charges: [] });
      if (path === "/api/teachers/lessons/lesson-1/outcome") { lesson.outcome = "completed"; return json(route, lesson); }
      if (path === "/api/teachers/lessons/lesson-1/note") return json(route, { id: "note-1", lesson: lesson.id, assignment: assignment.id, lesson_start_at: lesson.start_at, body: body.body, materials: [], updated_at: now.toISOString() });
      if (path.endsWith("/lesson-notes")) return json(route, { items: [] });
      if (path.endsWith("/materials") || path.endsWith("/pieces")) return json(route, { items: [] });
      if (path === "/api/teachers/assignments/assignment-1/practice-plan") {
        (body.create as { title: string }[]).forEach((task, index) => tasks.push({ id: `task-${index + 1}`, assignment: assignment.id, lesson: body.lesson, title: task.title, details: "", suggested_minutes: null, piece: null, material: null, status: "active", position: index + 1, created_at: now.toISOString() }));
        return json(route, { items: tasks });
      }
      if (path.endsWith("/practice-tasks")) return json(route, { items: tasks });
      if (path === "/api/learners/assignments/assignment-1/practice-sessions" && request.method() === "POST") { sessions.push(body as Session); return json(route, { id: "session-1", ...body, tasks: [], created_at: now.toISOString(), deletable: true }, 201); }
      if (path.endsWith("/practice-sessions")) return json(route, { items: [], page: 1, per_page: 20, total: 0 });
      if (path.endsWith("/practice-summary")) return json(route, { ...summaryOf(sessions, tasks), today: "2030-10-09", recent_days: [...new Set(sessions.map((session) => session.practiced_on))] });
      if (path === "/api/teachers/practice-summaries") return json(route, { items: [{ lesson: lesson.id, assignment: assignment.id, summary: summaryOf(sessions, tasks) }] });
      if (path.endsWith("/payment-due")) return json(route, { assignment: assignment.id, currency: "PLN", total_minor: 0, items: [], open_credit_minor: 0, next_forecast: null, recent_payments: [], instructions: null });
      return json(route, { code: "not_found", message: "Not found." }, 404);
    });
  }

  await teacherPage.goto("/teachers");
  await expect(teacherPage.getByText("Od ostatniej lekcji: brak zapisanych ćwiczeń.")).toBeVisible();
  await teacherPage.getByRole("button", { name: "Odbyta" }).click();
  await teacherPage.getByRole("button", { name: "Zadania na tydzień" }).click();
  await teacherPage.locator("#task-0-title").fill("Refren, tempo 70");
  await teacherPage.getByRole("button", { name: "Dodaj kolejne zadanie" }).click();
  await teacherPage.locator("#task-1-title").fill("Akordy C, G, a, F");
  await teacherPage.getByRole("button", { name: "Zapisz zadania" }).click();
  await expect(teacherPage.getByText("Zapisano zadania. Uczeń widzi je od razu.")).toBeVisible();
  await teacherPage.getByRole("button", { name: "Notatka po lekcji" }).click();
  await teacherPage.getByRole("textbox", { name: "Notatka po lekcji" }).fill("Refren coraz pewniej.");
  await teacherPage.getByRole("button", { name: "Zapisz notatkę" }).click();
  await expect(teacherPage.getByText("Zapisano notatkę. Uczeń widzi ją od razu.")).toBeVisible();
  const plan = writes.find((write) => write.path.endsWith("/practice-plan"));
  expect(plan?.body).toMatchObject({ lesson: "lesson-1", keep: [], done: [], archive: [], create: [{ title: "Refren, tempo 70" }, { title: "Akordy C, G, a, F" }] });

  await learnerPage.goto("/learners");
  await expect(learnerPage.getByRole("heading", { name: "Na ten tydzień" })).toBeVisible();
  await expect(learnerPage.getByText("Akordy C, G, a, F")).toBeVisible();
  await learnerPage.getByRole("button", { name: "Zapisz ćwiczenie" }).click();
  await learnerPage.getByLabel("Ile minut (opcjonalnie)").fill("20");
  await learnerPage.getByLabel("Komentarz dla nauczyciela (opcjonalnie)").fill("Takt 5 jeszcze nie wychodzi.");
  await learnerPage.getByRole("button", { name: "Zapisz", exact: true }).click();
  await expect(learnerPage.getByText("Od ostatniej lekcji: 1 dzień z ćwiczeniem, razem 20 min.")).toBeVisible();
  expect(sessions[0]).toEqual({ practiced_on: "2030-10-09", minutes: 20, tasks: ["task-1", "task-2"], comment: "Takt 5 jeszcze nie wychodzi." });

  await teacherPage.reload();
  await expect(teacherPage.getByText("Od ostatniej lekcji: 1 dzień ćwiczeń, 20 min.")).toBeVisible();
  await expect(teacherPage.getByText("„Takt 5 jeszcze nie wychodzi.”")).toBeVisible();
  await context.close();
});
