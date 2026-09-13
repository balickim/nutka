import { beforeEach, describe, expect, it, vi } from "vitest";

import { bookLesson, cancelLesson, fetchLearnerSlots, getSchedulingErrorMessage, ownCancellationCount, rescheduleLesson, SchedulingApiError } from "./scheduling";

describe("scheduling API client", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("loads slots with browser credentials", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ teacher: "t1", assignment: "a1", slots: [] }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await fetchLearnerSlots("a1");
    expect(fetchMock).toHaveBeenCalledWith("/api/learners/assignments/a1/slots", expect.objectContaining({ credentials: "include" }));
  });

  it("sends mutation intent and no learner duration override", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "lesson-1" }), { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);
    await bookLesson("a1", { start_at: "2026-01-15T12:00:00Z" });
    const [, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(options.headers).toMatchObject({ "X-Requested-With": "fetch", "Content-Type": "application/json" });
    expect(JSON.parse(String(options.body))).toEqual({ start_at: "2026-01-15T12:00:00Z" });
  });

  it("preserves stable API errors and mutation headers", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "conflict", message: "The interval conflicts." }), { status: 409 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(cancelLesson("learner", "lesson-1")).rejects.toMatchObject({ code: "conflict", status: 409 });
    await expect(cancelLesson("learner", "lesson-1")).rejects.toBeInstanceOf(SchedulingApiError);
    expect(fetchMock.mock.calls[0][1]).toEqual(expect.objectContaining({ credentials: "include" }));
  });

  it("maps stable API error codes to Polish UI copy without rendering server text", () => {
    expect(getSchedulingErrorMessage(new SchedulingApiError({ code: "conflict", message: "The interval conflicts." }, 409))).toContain("termin");
    expect(getSchedulingErrorMessage(new SchedulingApiError({ code: "unknown_code", message: "English server detail" }, 500))).toBe("Nie udało się wykonać operacji.");
  });

  it("reads only the matching initiator counter", () => {
    expect(ownCancellationCount({ teacher: 2, learner: 5 }, "teacher")).toBe(2);
    expect(ownCancellationCount({ teacher: 2, learner: 5 }, "learner")).toBe(5);
  });

  it("allows teacher duration-only reschedule payloads", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "lesson-1" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await rescheduleLesson("teacher", "lesson-1", { duration_minutes: 60 });
    expect(JSON.parse(String((fetchMock.mock.calls[0][1] as RequestInit).body))).toEqual({ duration_minutes: 60 });
  });
});
