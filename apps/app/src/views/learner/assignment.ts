// Selects the learner assignment that every learner screen shows from the requested identifier and the active assignments.

import type { Assignment } from "../../api/contracts";

export function activeAssignments(assignments: readonly Assignment[]): Assignment[] {
  return assignments.filter((assignment) => assignment.active);
}

export function selectAssignment(active: readonly Assignment[], requested: string | undefined): Assignment | undefined {
  return active.find((assignment) => assignment.id === requested) ?? active[0];
}
