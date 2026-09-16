import { beforeEach, describe, expect, it } from "vitest";

import { queryClient } from "../query/client";
import { queryKeys } from "../query/keys";
import { storePersonaSession } from "./session";

describe("persona cache purge for commercial resources", () => {
  beforeEach(() => {
    queryClient.clear();
    queryClient.setQueryData(queryKeys.policy("teacher"), { version: "one" });
    queryClient.setQueryData(
      queryKeys.commercialSummary("teacher", "teacher-1", "assignment-1"),
      { assignment: "assignment-1" },
    );
    queryClient.setQueryData(
      queryKeys.contractSeries("teacher", "assignment-1", "contract-1"),
      { contract: {} },
    );
    queryClient.setQueryData(queryKeys.financialWork("teacher", "teacher-1"), {
      charges: [],
      entries: [],
      credits: [],
    });
    queryClient.setQueryData(queryKeys.history("teacher", "teacher-1", "assignment-1"), {
      items: [],
      page: 1,
      per_page: 20,
      total: 0,
    });
    queryClient.setQueryData(queryKeys.policy("learner"), { version: "one" });
  });

  it("removes all teacher-owned commercial data and keeps learner data", async () => {
    await storePersonaSession(queryClient, "teacher", {
      kind: "unauthenticated",
    });
    expect(
      queryClient.getQueryData(queryKeys.policy("teacher")),
    ).toBeUndefined();
    expect(
      queryClient.getQueryData(
        queryKeys.commercialSummary("teacher", "teacher-1", "assignment-1"),
      ),
    ).toBeUndefined();
    expect(
      queryClient.getQueryData(
        queryKeys.contractSeries("teacher", "assignment-1", "contract-1"),
      ),
    ).toBeUndefined();
    expect(
      queryClient.getQueryData(queryKeys.financialWork("teacher", "teacher-1")),
    ).toBeUndefined();
    expect(
      queryClient.getQueryData(queryKeys.history("teacher", "teacher-1", "assignment-1")),
    ).toBeUndefined();
    expect(queryClient.getQueryData(queryKeys.policy("learner"))).toEqual({
      version: "one",
    });
  });
});
