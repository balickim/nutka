import { beforeEach, describe, expect, it } from "vitest";

import { createAppQueryClient } from "./client";
import { applyCacheEffect } from "./effects";
import {
  contractSeriesQuery,
  commercialSummaryQuery,
  businessPolicyQuery,
  financialWorkQuery,
  historyQuery,
} from "./commercial";
import { queryKeys, queryRules } from "./keys";

describe("commercial query ownership", () => {
  it("keeps policy and commercial data isolated by persona and account", () => {
    expect(businessPolicyQuery("teacher").queryKey).toEqual(
      queryKeys.policy("teacher"),
    );
    expect(businessPolicyQuery("teacher").queryKey).not.toEqual(
      businessPolicyQuery("learner").queryKey,
    );
    expect(
      commercialSummaryQuery("teacher", "teacher-1", "assignment-1").queryKey,
    ).not.toEqual(
      commercialSummaryQuery("learner", "learner-2", "assignment-1").queryKey,
    );
    expect(
      contractSeriesQuery("teacher", "teacher-1", "assignment-1", "contract-1")
        .queryKey,
    ).toEqual(
      queryKeys.contractSeries("teacher", "assignment-1", "contract-1"),
    );
    expect(
      historyQuery("teacher", "teacher-1", "assignment-1").queryKey,
    ).not.toEqual(
      historyQuery("learner", "learner-1", "assignment-1").queryKey,
    );
  });

  it("invalidates every dependent read for a compound plan mutation", async () => {
    const client = createAppQueryClient();
    const keys = [
      queryKeys.calendar("teacher", "teacher-1"),
      queryKeys.calendar("learner", "learner-1"),
      queryKeys.learnerSlots("learner-1", "assignment-1"),
      queryKeys.commercialSummary("teacher", "teacher-1", "assignment-1"),
      queryKeys.commercialSummary("learner", "learner-1", "assignment-1"),
      queryKeys.contractSeries("teacher", "assignment-1", "contract-1"),
      queryKeys.financialWork("teacher", "teacher-1", "assignment-1"),
      queryKeys.history("learner", "learner-1", "assignment-1"),
    ];
    keys.forEach((key) => client.setQueryData(key, { value: true }));
    await applyCacheEffect(client, queryRules.planWrite("assignment-1"));
    expect(keys.every((key) => client.getQueryState(key)?.isInvalidated)).toBe(
      true,
    );
  });

  it("does not invalidate a different assignment summary", async () => {
    const client = createAppQueryClient();
    const changed = queryKeys.commercialSummary("teacher", "teacher-1", "assignment-1");
    const untouched = queryKeys.commercialSummary("teacher", "teacher-1", "assignment-2");
    client.setQueryData(changed, { value: true });
    client.setQueryData(untouched, { value: true });
    await applyCacheEffect(client, queryRules.bookingWrite("assignment-1"));
    expect(client.getQueryState(changed)?.isInvalidated).toBe(true);
    expect(client.getQueryState(untouched)?.isInvalidated).toBe(false);
  });

  it("keeps teacher financial work distinct for each assignment", () => {
    expect(
      financialWorkQuery("teacher", "teacher-1", "assignment-1").queryKey,
    ).not.toEqual(
      financialWorkQuery("teacher", "teacher-1", "assignment-2").queryKey,
    );
  });
});
