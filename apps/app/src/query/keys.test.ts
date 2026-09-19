import { beforeEach, describe, expect, it } from "vitest";
import type { QueryClient } from "@tanstack/react-query";

import { createAppQueryClient } from "./client";
import { applyCacheEffect } from "./effects";
import { queryKeys, queryRules } from "./keys";

const teacherCalendar = queryKeys.calendar("teacher", "teacher-1");
const learnerCalendar = queryKeys.calendar("learner", "learner-1");
const firstSlots = queryKeys.learnerSlots("learner-1", "assignment-1");
const secondSlots = queryKeys.learnerSlots("learner-1", "assignment-2");
const seeded = [queryKeys.session("teacher"), queryKeys.session("learner"), teacherCalendar, learnerCalendar, firstSlots, secondSlots];

let client: QueryClient;

function seed() {
  seeded.forEach((key) => client.setQueryData(key, { seeded: true }));
}

function staleKeys(): string[] {
  return client
    .getQueryCache()
    .getAll()
    .filter((query) => query.state.isInvalidated)
    .map((query) => JSON.stringify(query.queryKey))
    .sort();
}

function cachedKeys(): string[] {
  return client.getQueryCache().getAll().map((query) => JSON.stringify(query.queryKey)).sort();
}

describe("query key registry", () => {
  beforeEach(() => { client = createAppQueryClient(); });

  it("builds a deterministic key for the same inputs", () => {
    expect(queryKeys.calendar("teacher", "teacher-1")).toEqual(queryKeys.calendar("teacher", "teacher-1"));
    expect(queryKeys.learnerSlots("learner-1", "assignment-1")).toEqual(["nutka", "learner", "slots", "learner-1", "assignment-1"]);
  });

  it("separates personas, accounts, and assignments", () => {
    expect(queryKeys.session("teacher")).not.toEqual(queryKeys.session("learner"));
    expect(queryKeys.calendar("teacher", "teacher-1")).not.toEqual(queryKeys.calendar("teacher", "teacher-2"));
    expect(firstSlots).not.toEqual(secondSlots);
    expect(queryKeys.learnerSlots("learner-2", "assignment-1")).not.toEqual(firstSlots);
  });

  it("nests package, contract, and month keys under the read they belong to", () => {
    const summary = queryKeys.commercialSummary("teacher", "teacher-1", "assignment-1");
    expect(queryKeys.assignmentPackages("teacher", "teacher-1", "assignment-1")).toEqual([...summary, "packages"]);
    expect(queryKeys.assignmentContracts("teacher", "teacher-1", "assignment-1")).toEqual([...summary, "contracts"]);
    expect(queryKeys.contractMonths("teacher", "assignment-1", "contract-1")).toEqual([...queryKeys.contractSeries("teacher", "assignment-1", "contract-1"), "months"]);
  });
});

describe("mutation cache rules", () => {
  beforeEach(() => { client = createAppQueryClient(); seed(); });

  it("refreshes both calendars and every learner slot after an availability write", async () => {
    await applyCacheEffect(client, queryRules.availabilityCommit());
    expect(staleKeys()).toEqual([teacherCalendar, learnerCalendar, firstSlots, secondSlots].map((key) => JSON.stringify(key)).sort());
  });

  it("refreshes the packages and contracts of the changed assignment after a plan write", async () => {
    const packages = queryKeys.assignmentPackages("teacher", "teacher-1", "assignment-1");
    const contracts = queryKeys.assignmentContracts("teacher", "teacher-1", "assignment-1");
    const other = queryKeys.assignmentPackages("teacher", "teacher-1", "assignment-2");
    [packages, contracts, other].forEach((key) => client.setQueryData(key, { seeded: true }));
    await applyCacheEffect(client, queryRules.planWrite("assignment-1"));
    expect(staleKeys()).toContain(JSON.stringify(packages));
    expect(staleKeys()).toContain(JSON.stringify(contracts));
    expect(staleKeys()).not.toContain(JSON.stringify(other));
  });

  it("refreshes both calendars and only the changed assignment after an assignment write", async () => {
    await applyCacheEffect(client, queryRules.assignmentWrite("assignment-2"));
    expect(staleKeys()).toEqual([teacherCalendar, learnerCalendar, secondSlots].map((key) => JSON.stringify(key)).sort());
  });

  it("refreshes teacher and learner materials of only the changed assignment after a material write", async () => {
    const teacherMaterials = queryKeys.materials("teacher", "teacher-1", "assignment-1");
    const learnerMaterials = queryKeys.materials("learner", "learner-1", "assignment-1");
    const other = queryKeys.materials("teacher", "teacher-1", "assignment-2");
    [teacherMaterials, learnerMaterials, other].forEach((key) => client.setQueryData(key, { seeded: true }));
    await applyCacheEffect(client, queryRules.materialWrite("assignment-1"));
    expect(staleKeys()).toEqual([teacherMaterials, learnerMaterials].map((key) => JSON.stringify(key)).sort());
  });

  it("refreshes both calendars and every learner slot after a lesson write", async () => {
    await applyCacheEffect(client, queryRules.lessonWrite());
    expect(staleKeys()).toEqual([teacherCalendar, learnerCalendar, firstSlots, secondSlots].map((key) => JSON.stringify(key)).sort());
  });

  it("removes the owned data of the cleared persona and keeps every other persona key", async () => {
    await applyCacheEffect(client, queryRules.personaCleared("teacher"));
    const kept = [queryKeys.session("teacher"), queryKeys.session("learner"), learnerCalendar, firstSlots, secondSlots];
    expect(cachedKeys()).toEqual(kept.map((key) => JSON.stringify(key)).sort());
  });
});
