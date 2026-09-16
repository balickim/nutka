import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { queryClient } from "../query/client";
import { queryKeys } from "../query/keys";
import { ensurePersonaSession, storePersonaSession, type SessionResult } from "./session";

const learnerCalendar = queryKeys.calendar("learner", "learner-1");
const teacherCalendar = queryKeys.calendar("teacher", "teacher-1");
const learnerSlots = queryKeys.learnerSlots("learner-1", "assignment-1");

function session(role: "teacher" | "learner"): SessionResult | undefined {
  return queryClient.getQueryData(queryKeys.session(role));
}

function authenticated(id: string, expiresAt?: string): SessionResult {
  return { kind: "authenticated", record: { id, email: `${id}@example.test` }, expiresAt };
}

describe("persona session query", () => {
  beforeEach(() => { queryClient.clear(); vi.restoreAllMocks(); });

  it("shares one request between a route guard and a rendered consumer", async () => {
    const mock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ record: { id: "learner-1", email: "learner@example.test" } }), { status: 200 }));
    vi.stubGlobal("fetch", mock);
    const [guard, consumer] = await Promise.all([ensurePersonaSession("learner"), ensurePersonaSession("learner")]);
    expect(mock).toHaveBeenCalledTimes(1);
    expect(guard).toEqual(consumer);
    expect(guard.kind).toBe("authenticated");
  });

  it("reads a 401 as an unauthenticated session instead of an error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 401 })));
    await expect(ensurePersonaSession("teacher")).resolves.toEqual({ kind: "unauthenticated" });
  });

  it("retries a 5xx bootstrap once and then reports the failure", async () => {
    const mock = vi.fn().mockResolvedValue(new Response(null, { status: 503 }));
    vi.stubGlobal("fetch", mock);
    await expect(ensurePersonaSession("teacher")).rejects.toMatchObject({ status: 503 });
    expect(mock).toHaveBeenCalledTimes(2);
  });
});

describe("persona cache lifecycle", () => {
  beforeEach(() => {
    queryClient.clear();
    queryClient.setQueryData(learnerCalendar, { seeded: true });
    queryClient.setQueryData(learnerSlots, { seeded: true });
    queryClient.setQueryData(teacherCalendar, { seeded: true });
    queryClient.setQueryData(queryKeys.session("teacher"), authenticated("teacher-1"));
  });
  afterEach(() => { vi.useRealTimers(); });

  it("drops the previous account cache before exposing a replacement session", async () => {
    await storePersonaSession(queryClient, "learner", authenticated("learner-2"));
    expect(queryClient.getQueryData(learnerCalendar)).toBeUndefined();
    expect(queryClient.getQueryData(learnerSlots)).toBeUndefined();
    expect(session("learner")).toEqual(authenticated("learner-2"));
  });

  it("clears only the persona that logged out", async () => {
    await storePersonaSession(queryClient, "teacher", { kind: "unauthenticated" });
    expect(queryClient.getQueryData(teacherCalendar)).toBeUndefined();
    expect(session("teacher")).toEqual({ kind: "unauthenticated" });
    expect(queryClient.getQueryData(learnerCalendar)).toEqual({ seeded: true });
  });

  it("clears the persona cache when the reported expiry passes", async () => {
    vi.useFakeTimers();
    await storePersonaSession(queryClient, "learner", authenticated("learner-1", new Date(Date.now() - 1000).toISOString()));
    await vi.advanceTimersByTimeAsync(1);
    expect(session("learner")).toEqual({ kind: "unauthenticated" });
    expect(queryClient.getQueryData(learnerCalendar)).toBeUndefined();
  });

  it("clears the persona cache when another tab reports a logout", async () => {
    await storePersonaSession(queryClient, "learner", authenticated("learner-1"));
    const otherTab = new BroadcastChannel("nutka-learner-auth");
    otherTab.postMessage({ type: "logout" });
    await vi.waitFor(() => expect(session("learner")).toEqual({ kind: "unauthenticated" }));
    otherTab.close();
    expect(session("teacher")).toEqual(authenticated("teacher-1"));
  });
});
