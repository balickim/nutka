import { describe, expect, it } from "vitest";

import { paymentFacts } from "./payment-facts";

describe("paymentFacts", () => {
  it("returns nothing when every payment value is zero", () => {
    expect(paymentFacts({ pending: 0, intentionally_unpaid: 0, overdue: 0, credit_minor: 0, currency: "PLN" })).toEqual([]);
  });
  it("keeps only the non-zero values", () => {
    const facts = paymentFacts({ pending: 1, intentionally_unpaid: 0, overdue: 0, credit_minor: 5000, currency: "PLN" });
    expect(facts).toHaveLength(2);
    expect(facts[0]).toBe("Płatności do rozliczenia: 1.");
    expect(facts[1]).toBe("Nadpłata do wykorzystania: 50,00 zł.");
  });
});
