import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  bookFlexibleLesson,
  fetchBusinessPolicy,
  fetchHistory,
  recordLessonOutcome,
  rescheduleCommercialLesson,
} from "./commercial";

describe("commercial API contracts", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("reads policy through the matching authenticated persona route", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(
          JSON.stringify({
            version: "v1",
            currency: "PLN",
            ad_hoc_price_minor: 8000,
            package_price_minor: 26000,
            regular_lesson_price_minor: 5000,
            lesson_duration_minutes: 45,
            start_grid_minutes: 15,
            participant_buffer_minutes: 5,
            learner_booking_minimum_hours: 24,
            learner_change_cutoff_hours: 24,
            booking_horizon_days: 14,
            package_token_count: 4,
            package_validity_days: 60,
            teacher_cancellation_extension_days: 7,
            contract_monthly_reschedules: 1,
            contract_free_cancellations: 2,
            contract_replacement_deadline_days: 30,
            monthly_payment_due_day: 5,
            contract_end_month: 6,
            contract_end_day: 30,
          }),
          { status: 200 },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);
    const policy = await fetchBusinessPolicy("learner");
    expect(fetchMock.mock.calls[0][0]).toBe("/api/learners/business-policy");
    expect(policy.contract_replacement_deadline_days).toBe(30);
    expect(policy).not.toHaveProperty("contract_replacement_days");
  });

  it("uses the teacher short-notice field without sending a duration or caller identity", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ id: "lesson-1" }), { status: 201 }),
      );
    vi.stubGlobal("fetch", fetchMock);
    await bookFlexibleLesson("teacher", "assignment/1", {
      start_at: "2030-01-07T09:00:00Z",
      confirm_short_notice: true,
    });
    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/teachers/assignments/assignment%2F1/book");
    expect(JSON.parse(String(options.body))).toEqual({
      start_at: "2030-01-07T09:00:00Z",
      confirm_short_notice: true,
    });
  });

  it("uses the lifecycle replacement contract and POST transport", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ id: "lesson-1" }), { status: 200 }),
      );
    vi.stubGlobal("fetch", fetchMock);
    await rescheduleCommercialLesson("learner", "lesson-1", {
      start_at: "2030-01-08T09:00:00Z",
    });
    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/learners/lessons/lesson-1/reschedule");
    expect(options.method).toBe("POST");
    expect(JSON.parse(String(options.body))).toEqual({
      start_at: "2030-01-08T09:00:00Z",
    });
  });

  it("uses the backend-redacted learner history page without client-side filtering", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          items: [
            {
              id: "event-1",
              event_type: "lesson_cancelled",
            },
          ],
          page: 1,
          per_page: 20,
          total: 1,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const result = await fetchHistory("learner", "assignment-1");
    expect(result).toEqual({
      items: [{ id: "event-1", event_type: "lesson_cancelled" }],
      page: 1,
      per_page: 20,
      total: 1,
    });
  });

  it("posts explicit outcome values only", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ id: "lesson-1" }), { status: 200 }),
      );
    vi.stubGlobal("fetch", fetchMock);
    await recordLessonOutcome("lesson-1", { outcome: "learner_no_show" });
    expect(
      JSON.parse(
        String((fetchMock.mock.calls[0] as [string, RequestInit])[1].body),
      ),
    ).toEqual({ outcome: "learner_no_show" });
  });
});
