import { beforeEach, describe, expect, it, vi } from "vitest";
import type { QueryClient } from "@tanstack/react-query";

import { bookLesson } from "../api/scheduling";
import { createAppQueryClient } from "./client";
import { queryKeys } from "./keys";
import { learnerCalendarQuery, learnerSlotsQuery, teacherCalendarQuery } from "./scheduling";

let client: QueryClient;

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status });
}

describe("scheduling query options", () => {
  beforeEach(() => { client = createAppQueryClient(); vi.restoreAllMocks(); });

  it("keys owned data by persona, account, and assignment", () => {
    expect(teacherCalendarQuery("teacher-1").queryKey).toEqual(queryKeys.calendar("teacher", "teacher-1"));
    expect(learnerCalendarQuery("learner-1").queryKey).toEqual(queryKeys.calendar("learner", "learner-1"));
    expect(learnerSlotsQuery("learner-1", "assignment-1").queryKey).toEqual(queryKeys.learnerSlots("learner-1", "assignment-1"));
  });

  it("stays disabled until the account is authenticated", () => {
    expect(teacherCalendarQuery(undefined).enabled).toBe(false);
    expect(learnerSlotsQuery(undefined, "assignment-1").enabled).toBe(false);
    expect(learnerSlotsQuery("learner-1", "assignment-1").enabled).toBe(true);
  });

  it("retains the cached calendar until a refresh settles", async () => {
    let release = () => {};
    const mock = vi.fn()
      .mockResolvedValueOnce(json({ lessons: [] }))
      .mockImplementationOnce(() => new Promise((resolve) => { release = () => resolve(json({ lessons: [{ id: "lesson-1" }] })); }));
    vi.stubGlobal("fetch", mock);
    const options = teacherCalendarQuery("teacher-1");
    await client.fetchQuery(options);
    const refresh = client.fetchQuery({ ...options, staleTime: 0 });
    await vi.waitFor(() => expect(mock).toHaveBeenCalledTimes(2));
    expect(client.getQueryData(options.queryKey)).toEqual({ lessons: [] });
    release();
    await refresh;
    expect(client.getQueryData(options.queryKey)).toEqual({ lessons: [{ id: "lesson-1" }] });
  });

  it("does not repeat a deterministic read failure", async () => {
    const mock = vi.fn().mockResolvedValue(json({ code: "unauthenticated", message: "Authentication is required." }, 401));
    vi.stubGlobal("fetch", mock);
    await expect(client.fetchQuery(teacherCalendarQuery("teacher-1"))).rejects.toMatchObject({ code: "unauthenticated" });
    expect(mock).toHaveBeenCalledTimes(1);
  });

  it("does not repeat a rejected scheduling write", async () => {
    const mock = vi.fn().mockResolvedValue(json({ code: "conflict", message: "The interval conflicts." }, 409));
    vi.stubGlobal("fetch", mock);
    const mutation = client.getMutationCache().build(client, { mutationFn: () => bookLesson("assignment-1", { start_at: "2026-01-15T12:00:00Z" }) });
    await expect(mutation.execute(undefined)).rejects.toMatchObject({ code: "conflict" });
    expect(mock).toHaveBeenCalledTimes(1);
  });
});
