import { describe, expect, it } from "vitest";

import { loginSchema } from "./validation";

describe("login validation", () => {
  it("requires a valid email and a non-empty password", () => {
    expect(loginSchema.safeParse({ email: "learner@example.test", password: "secret" }).success).toBe(true);
    expect(loginSchema.safeParse({ email: "not-an-email", password: "secret" }).success).toBe(false);
    expect(loginSchema.safeParse({ email: "learner@example.test", password: "" }).success).toBe(false);
  });
});
