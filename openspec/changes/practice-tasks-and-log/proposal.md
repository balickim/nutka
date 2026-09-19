## Why

Adult learners forget what to practice between lessons, and the teacher does not know what happened during the week. The landing page promises a plan and steady progress. The app has lesson notes and pieces, but no practice tasks and no record that the learner practiced.

## What Changes

- Add practice tasks per assignment with a title, details, suggested minutes, an optional piece and material, and the status `active`, `done`, or `archived`.
- Add one teacher plan write that keeps, completes, and archives active tasks and creates new tasks in one transaction.
- Add learner practice sessions for a local day up to 14 days back, with optional minutes, tasks, and a comment. The learner deletes an own session within 7 days.
- Add a practice summary since the last started lesson and a teacher batch summary for the lessons of one day.
- Add the learner Ćwiczenia screen, tasks on Start, a tasks section in the teacher after-lesson flow, a summary line on the Today card, and a teacher Ćwiczenia tab.

## Capabilities

### New Capabilities

- `practice-tasks-and-log`: Practice tasks, learner practice sessions, and practice summaries.

### Modified Capabilities

None.

## Impact

- Backend: `practice_tasks` and `practice_sessions` collections, `internal/practice`, `internal/practiceapi`.
- Frontend: `api/practice.ts`, practice queries and cache rules, route `/learners/practice`, learner Start, teacher Today card, teacher learner view.
- Docs: `docs/api/practice.md`, `docs/README.md`, `docs/constitutions/learner-content.md`, `docs/constitutions/frontend-view-states.md`.
