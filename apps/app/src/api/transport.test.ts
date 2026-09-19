import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiRequestError, apiRequest, isRetryableFailure } from "./transport";

function stubFetch(response: Response | Error) {
  const mock = vi.fn().mockImplementation(() => (response instanceof Error ? Promise.reject(response) : Promise.resolve(response)));
  vi.stubGlobal("fetch", mock);
  return mock;
}

describe("API transport boundary", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("sends cookies and forwards the abort signal of a cancelled read", async () => {
    const mock = stubFetch(new Response(JSON.stringify({ slots: [] }), { status: 200 }));
    const controller = new AbortController();
    await apiRequest("/api/learners/assignments/a1/slots", { signal: controller.signal });
    const [url, init] = mock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/learners/assignments/a1/slots");
    expect(init.credentials).toBe("include");
    expect(init.signal).toBe(controller.signal);
    expect(init.headers).not.toMatchObject({ "X-Requested-With": "fetch" });
  });

  it("sends the mutation intent header and a JSON body", async () => {
    const mock = stubFetch(new Response(JSON.stringify({ id: "lesson-1" }), { status: 201 }));
    await apiRequest("/api/learners/assignments/a1/book", { method: "POST", body: { start_at: "2026-01-15T12:00:00Z" } });
    const [, init] = mock.mock.calls[0] as [string, RequestInit];
    expect(init.headers).toMatchObject({ "X-Requested-With": "fetch", "Content-Type": "application/json" });
    expect(JSON.parse(String(init.body))).toEqual({ start_at: "2026-01-15T12:00:00Z" });
  });

  it("sends a multipart body without a JSON content type so the browser sets the boundary", async () => {
    const mock = stubFetch(new Response(JSON.stringify({ id: "m1" }), { status: 201 }));
    const form = new FormData();
    form.append("title", "Gamy");
    await apiRequest("/api/teachers/assignments/a1/materials", { method: "POST", body: form });
    const [, init] = mock.mock.calls[0] as [string, RequestInit];
    expect(init.body).toBe(form);
    expect(init.headers).toEqual({ "X-Requested-With": "fetch" });
  });

  it("returns decoded JSON and no body for an empty response", async () => {
    stubFetch(new Response(JSON.stringify({ teacher: "t1" }), { status: 200 }));
    await expect(apiRequest("/api/teachers/calendar")).resolves.toEqual({ teacher: "t1" });
    stubFetch(new Response(null, { status: 204 }));
    await expect(apiRequest("/api/teachers/availability/rules/r1", { method: "DELETE" })).resolves.toBeUndefined();
  });

  it("keeps the stable error code of a 4xx response", async () => {
    stubFetch(new Response(JSON.stringify({ code: "conflict", message: "The interval conflicts." }), { status: 409 }));
    await expect(apiRequest("/api/learners/lessons/l1/cancel", { method: "POST" })).rejects.toMatchObject({ code: "conflict", status: 409 });
  });

  it("uses the generic error for a non-JSON 5xx response", async () => {
    stubFetch(new Response("gateway", { status: 502 }));
    await expect(apiRequest("/api/teachers/calendar")).rejects.toMatchObject({ code: "request_failed", status: 502 });
  });

  it("propagates a network failure unchanged", async () => {
    stubFetch(new Error("offline"));
    await expect(apiRequest("/api/teachers/calendar")).rejects.toThrow("offline");
  });

  it("treats only deterministic 4xx responses as final", () => {
    expect(isRetryableFailure(new ApiRequestError({ code: "conflict", message: "" }, 409))).toBe(false);
    expect(isRetryableFailure(new ApiRequestError({ code: "request_failed", message: "" }, 503))).toBe(true);
    expect(isRetryableFailure(new Error("offline"))).toBe(true);
  });
});
