import { describe, expect, it } from "vitest";

import { formatMoney, parseMajor, toMajorInput } from "./money";

describe("money", () => {
  it("parses a major-unit value with either separator", () => {
    expect(parseMajor("50")).toBe(5000);
    expect(parseMajor("50,5")).toBe(5050);
    expect(parseMajor(" 260.00 ")).toBe(26000);
  });

  it("rejects a value the API cannot represent", () => {
    expect(parseMajor("")).toBeNull();
    expect(parseMajor("50.005")).toBeNull();
    expect(parseMajor("abc")).toBeNull();
  });

  it("formats zero, fractions, and negative credit", () => {
    expect(formatMoney(0, "PLN")).toBe("0,00 PLN");
    expect(formatMoney(5005, "PLN")).toBe("50,05 PLN");
    expect(formatMoney(-2500, "PLN")).toBe("-25,00 PLN");
  });

  it("round-trips through the input representation", () => {
    expect(parseMajor(toMajorInput(26000))).toBe(26000);
  });
});
