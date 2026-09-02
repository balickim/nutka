import { beforeEach, describe, expect, it, vi } from "vitest";

import { bootstrapAuth, classifyAuthError, getAuthState, logout } from "./auth";

describe("auth helpers", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("classifies HTTP 4xx responses as generic credential failures", () => {
    expect(classifyAuthError({ status: 400 })).toBe("invalid-credentials");
    expect(classifyAuthError({ status: 401 })).toBe("invalid-credentials");
    expect(classifyAuthError({ status: 503 })).toBe("retryable");
    expect(classifyAuthError(new Error("offline"))).toBe("retryable");
  });

  it("single-flights an unauthenticated bootstrap and clears local state", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 401 }));
    vi.stubGlobal("fetch", fetchMock);
    const first = bootstrapAuth();
    const second = bootstrapAuth();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const result = await first;
    await second;
    expect(result.kind).toBe("unauthenticated");
    expect(getAuthState().record).toBeNull();
  });

  it("clears local state even when logout cannot reach the server", async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error("offline"));
    vi.stubGlobal("fetch", fetchMock);
    await logout();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/auth/logout",
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
    expect(getAuthState().record).toBeNull();
  });
});
