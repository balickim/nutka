import { describe, expect, it } from "vitest";

import type { Assignment } from "../../api/contracts";
import { activeAssignments, selectAssignment } from "./assignment";

const assignment = (id: string, active = true): Assignment => ({ id, teacher: `t-${id}`, learner: "l", active });

describe("selectAssignment", () => {
  const active = activeAssignments([assignment("a"), assignment("b"), assignment("off", false)]);

  it("returns the requested active assignment", () => {
    expect(selectAssignment(active, "b")?.id).toBe("b");
  });
  it("falls back to the first active assignment for missing, foreign, or inactive values", () => {
    expect(selectAssignment(active, undefined)?.id).toBe("a");
    expect(selectAssignment(active, "foreign")?.id).toBe("a");
    expect(selectAssignment(active, "off")?.id).toBe("a");
  });
  it("returns nothing without an active assignment", () => {
    expect(selectAssignment([], "a")).toBeUndefined();
  });
});
