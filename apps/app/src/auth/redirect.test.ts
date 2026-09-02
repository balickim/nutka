import { describe, expect, it } from "vitest";

import { getSafeRedirect } from "./redirect";

describe("getSafeRedirect", () => {
  it("accepts a same-origin path with query and hash", () => {
    expect(getSafeRedirect("/lesson/1?week=2#practice")).toBe("/lesson/1?week=2#practice");
  });
  it("rejects absolute and protocol-relative URLs", () => {
    expect(getSafeRedirect("https://evil.example/login")).toBe("/");
    expect(getSafeRedirect("//evil.example/login")).toBe("/");
  });
  it("rejects missing and backslash redirects", () => {
    expect(getSafeRedirect(undefined)).toBe("/");
    expect(getSafeRedirect("/\\evil.example")).toBe("/");
  });
});
