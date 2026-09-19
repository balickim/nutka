import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiRequestError } from "./transport";
import { bookLesson, getSchedulingErrorMessage, rescheduleLesson } from "./scheduling";

describe("scheduling endpoints", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("books without a learner duration override", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "lesson-1" }), { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);
    await bookLesson("a1", { start_at: "2026-01-15T12:00:00Z" });
    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/learners/assignments/a1/book");
    expect(JSON.parse(String(options.body))).toEqual({ start_at: "2026-01-15T12:00:00Z" });
  });

  it("uses the fixed-duration reschedule payload", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "lesson-1" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await rescheduleLesson("teacher", "lesson-1", { start_at: "2026-01-16T12:00:00Z" });
    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/teachers/lessons/lesson-1/reschedule");
    expect(options.method).toBe("POST");
    expect(JSON.parse(String(options.body))).toEqual({ start_at: "2026-01-16T12:00:00Z" });
  });

  it("maps stable API error codes to Polish UI copy without rendering server text", () => {
    expect(getSchedulingErrorMessage(new ApiRequestError({ code: "conflict", message: "The interval conflicts." }, 409))).toContain("termin");
    expect(getSchedulingErrorMessage(new ApiRequestError({ code: "unknown_code", message: "English server detail" }, 500))).toBe("Coś poszło nie tak. Spróbuj ponownie.");
  });
});
