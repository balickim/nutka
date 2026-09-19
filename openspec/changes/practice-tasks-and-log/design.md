## Context

See `proposal.md` for motivation and `specs/practice-tasks-and-log/spec.md` for behavior. `docs/constitutions/learner-content.md` defines the shared rules for assignment-scoped learner content. Lesson notes (C) and pieces (D) exist.

## Goals / Non-Goals

**Goals:**

- The learner sees what to practice and records a session in one short form.
- The teacher sees what happened since the last lesson before the next lesson, with one request for the Today screen.

**Non-Goals:**

- Points, streaks, badges, or rankings.
- Session edits. The learner deletes and adds again.
- A week entity. The active tasks are the plan.

## Decisions

### Local practice day

`practice_sessions.practiced_on` stores a local date `YYYY-MM-DD` in the teacher timezone, not an instant. A practice session belongs to a day, and a date avoids a wrong day at the Warsaw boundary. The server derives today from its clock and the teacher timezone. It accepts a date from today back to 14 days before today.

### Tasks

`practice_tasks` has `assignment`, optional `lesson`, `title` (max 200), `details` (max 1000, plain text), optional `suggested_minutes` (1 to 120), optional `piece` and `material` of the same assignment, `status`, and `position`. The deletion of a piece or a material clears the link.

### Plan write

`PUT /api/teachers/assignments/{id}/practice-plan` takes `lesson`, `keep`, `done`, `archive`, and `create`. The server checks that every identifier belongs to the assignment and appears in one list only. It writes all changes in one PocketBase transaction. The new positions follow `keep` in request order, then the other active tasks in stored order, then `create`. An active task that the request does not name stays active.

### Summary window

The window starts on the local day of the latest scheduled lesson of the assignment whose start is not after the reference instant. Without such a lesson, the window starts 6 days before today, so it covers 7 days. The window includes the lesson day, because learners often practice in the evening after the lesson. The teacher batch summary uses, for each lesson of the date, the latest earlier lesson of that assignment.

### Pure rules in `internal/practice`

`ValidateTask`, `CheckDay`, `ValidateSession`, `LearnerMayDelete`, `WindowStart`, `Summarize`, and `RecentDays` have no PocketBase dependency and have table tests.

### Cache rules

`practiceTaskWrite(assignmentId)` invalidates tasks and summaries of the assignment for both personas and the teacher day summaries. `practiceSessionWrite(assignmentId)` invalidates sessions and summaries of the assignment for both personas and the teacher day summaries.

## Risks / Trade-offs

- [A teacher changes the timezone] → Stored days stay. The window moves at most one day.
- [Many lessons on one day] → The batch route reads lessons of that day once and sessions per assignment once.
