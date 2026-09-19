import { expect, test } from "@playwright/test";
import type { Route } from "@playwright/test";

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
const calendar = { assignments: [assignment], availability_rules: [], availability_exceptions: [], commercial_summaries: [], near_term_lessons: [], later_contract_lessons: [], unresolved_work: { awaiting_outcome: 0, pending_settlement: 0, unpaid_charges: 0, overdue_charges: 0 } };

type Piece = { id: string; assignment: string; title: string; artist: string; status: string; status_changed_at: string; proposed_by: string; material_count: number; latest_material_at: string | null; created_at: string };

const piece = (id: string, title: string, status: string, proposedBy: string): Piece => ({ id, assignment: assignment.id, title, artist: "", status, status_changed_at: "2030-10-01T10:00:00Z", proposed_by: proposedBy, material_count: 0, latest_material_at: null, created_at: "2030-10-01T10:00:00Z" });
const material = (id: string, title: string, created: string, pieceId: string | null) => ({ id, assignment: assignment.id, title, body: `<p>${title}</p>`, piece: pieceId, attachments: [], created_at: created });

async function json(route: Route, value: unknown, status = 200) {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(value) });
}

test("a learner wish reaches the teacher, and the started piece shows its versions to the learner", async ({ browser }) => {
  const pieces: Piece[] = [piece("piece-1", "Hallelujah", "learning", "teacher")];
  const materials = [material("m-1", "Hallelujah – łatwa", "2030-10-01T10:00:00Z", "piece-1"), material("m-2", "Hallelujah – z arpeggio", "2030-10-08T10:00:00Z", "piece-1"), material("m-3", "Gama C-dur", "2030-10-02T10:00:00Z", null)];
  const writes: { method: string; path: string; body: unknown; intent?: string }[] = [];
  const context = await browser.newContext();
  const learnerPage = await context.newPage();
  const teacherPage = await context.newPage();
  for (const page of [learnerPage, teacherPage]) {
    await page.route(apiRequestPattern, async (route) => {
      const request = route.request();
      const path = new URL(request.url()).pathname;
      const persona = path.startsWith("/api/teachers/") ? "teachers" : "learners";
      if (path === "/api/health") return json(route, {});
      if (path === `/api/${persona}/auth/me`) return json(route, { record: persona === "teachers" ? teacher : learner });
      if (path === `/api/${persona}/business-policy`) return json(route, policy);
      if (path === `/api/${persona}/calendar`) return json(route, calendar);
      if (path === `/api/${persona}/assignments/assignment-1/pieces` && request.method() === "GET") return json(route, { items: pieces });
      if (path === `/api/${persona}/assignments/assignment-1/materials`) return json(route, { items: materials });
      if (path === "/api/learners/assignments/assignment-1/pieces") {
        const body = request.postDataJSON() as { title: string; artist: string };
        writes.push({ method: "POST", path, body, intent: request.headers()["x-requested-with"] });
        const created = { ...piece("piece-2", body.title, "wish", "learner"), artist: body.artist };
        pieces.push(created);
        return json(route, created, 201);
      }
      if (path === "/api/teachers/pieces/piece-2" && request.method() === "PATCH") {
        writes.push({ method: "PATCH", path, body: request.postDataJSON() });
        pieces[1] = { ...pieces[1], status: "learning" };
        return json(route, pieces[1]);
      }
      if (path === "/api/teachers/unresolved-work") return json(route, { awaiting_outcome: [], pending_settlements: [], unpaid_charges: [], overdue_charges: [] });
      if (path === "/api/teachers/financial-work") return json(route, { charges: [], entries: [], credits: [] });
      return json(route, { code: "not_found", message: "Not found." }, 404);
    });
  }

  await learnerPage.goto("/learners/pieces");
  await expect(learnerPage.getByRole("heading", { name: "Uczę się" })).toBeVisible();
  await learnerPage.getByText("Pokaż 2 opracowania").click();
  const versions = learnerPage.locator(".piece-versions .material-card");
  await expect(versions.first()).toContainText("Wersja 2");
  await expect(versions.first()).toContainText("Hallelujah – z arpeggio");
  await expect(learnerPage.getByRole("heading", { name: "Inne materiały" })).toBeVisible();
  await expect(learnerPage.getByRole("heading", { name: "Gama C-dur" })).toBeVisible();

  await learnerPage.getByLabel("Tytuł utworu").fill("Perfect");
  await learnerPage.getByLabel("Wykonawca lub kompozytor (opcjonalnie)").fill("Ed Sheeran");
  await learnerPage.getByRole("button", { name: "Dodaj do listy" }).click();
  await expect(learnerPage.locator(".wish-list")).toContainText("Perfect · Ed Sheeran");
  expect(writes[0]).toEqual({ method: "POST", path: "/api/learners/assignments/assignment-1/pieces", body: { title: "Perfect", artist: "Ed Sheeran" }, intent: "fetch" });

  await teacherPage.goto("/teachers/students/assignment-1");
  await teacherPage.getByRole("tab", { name: "Utwory" }).click();
  await expect(teacherPage.getByRole("heading", { name: "Życzenia ucznia" })).toBeVisible();
  await teacherPage.getByRole("button", { name: "Zacznij naukę" }).click();
  await expect(teacherPage.getByRole("heading", { name: "Życzenia ucznia" })).toHaveCount(0);
  expect(writes[1]).toEqual({ method: "PATCH", path: "/api/teachers/pieces/piece-2", body: { status: "learning" } });

  await learnerPage.reload();
  await expect(learnerPage.locator(".piece-group").filter({ hasText: "Uczę się" })).toContainText("Perfect");
  await expect(learnerPage.locator(".wish-list")).toHaveCount(0);
  await context.close();
});
