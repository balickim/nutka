import { beforeEach, describe, expect, it, vi } from "vitest";

import { learnerAuth } from "./auth";
import { getAuthState, login, logout, teacherAuth } from "./teacher";

describe("teacher auth realm", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("uses the teacher collection without exposing the native token", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      record: { id: "teacher-1", email: "teacher@example.test", name: "Teacher" },
      token: "must-not-enter-client-state",
    }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);
    await login("teacher@example.test", "secret");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/collections/teachers/auth-with-password",
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
    expect(getAuthState().record).toMatchObject({ id: "teacher-1" });
    expect(JSON.stringify(getAuthState())).not.toContain("must-not-enter-client-state");
    await logout();
  });

  it("clears only the teacher realm when teacher logout completes", async () => {
    const fetchMock = vi.fn().mockImplementation((input: string) => {
      if (input.includes("auth-with-password")) {
        const teacher = input.includes("/teachers/");
        return Promise.resolve(new Response(JSON.stringify({
          record: {
            id: teacher ? "teacher-1" : "learner-1",
            email: teacher ? "teacher@example.test" : "learner@example.test",
          },
        }), { status: 200 }));
      }
      return Promise.resolve(new Response(JSON.stringify({ status: "ok" }), { status: 200 }));
    });
    vi.stubGlobal("fetch", fetchMock);

    await learnerAuth.login("learner@example.test", "secret");
    await teacherAuth.login("teacher@example.test", "secret");
    await teacherAuth.logout();

    expect(teacherAuth.getState().record).toBeNull();
    expect(learnerAuth.getState().record).toMatchObject({ id: "learner-1" });

    await learnerAuth.logout();
    expect(learnerAuth.getState().record).toBeNull();
  });
});
