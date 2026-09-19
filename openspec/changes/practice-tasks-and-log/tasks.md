## 1. Backend

- [x] 1.1 Add `internal/practice` with names, task and session rules, day window, summary, and recent days, and table tests
- [x] 1.2 Add the migration for `practice_tasks` and `practice_sessions`
- [x] 1.3 Add `internal/practiceapi` with task list, plan, and task update routes
- [x] 1.4 Add session create, delete, and list routes
- [x] 1.5 Add the assignment summary and the teacher day summaries, and register the package in `main.go`
- [x] 1.6 Add HTTP tests: future and old day, foreign task in session, late delete, inactive assignment, atomic plan, summary window
- [x] 1.7 Run `go build ./...` and `go test ./...`

## 2. Frontend data

- [x] 2.1 Add `api/practice.ts`
- [x] 2.2 Add query keys, queries, cache rules, and `keys.test.ts` cases
- [x] 2.3 Add pure helpers for day choices and the 4-week grid with tests

## 3. Learner screens

- [x] 3.1 Add route `/learners/practice` and the Ćwiczenia navigation item
- [x] 3.2 Add the task list, the session form, the session list with delete, and the 4-week grid
- [x] 3.3 Show active tasks and the session button on Start

## 4. Teacher screens

- [x] 4.1 Add the tasks section to the after-lesson flow
- [x] 4.2 Show the practice summary on the Today card
- [x] 4.3 Add the Ćwiczenia tab with tasks, task status changes, and sessions

## 5. Tests, docs, and verification

- [x] 5.1 Add a Playwright flow: the teacher saves 2 tasks, the learner sees them on Start and records practice, the teacher sees the summary on Today
- [x] 5.2 Add `docs/api/practice.md` and update `docs/README.md`, `learner-content.md`, and `frontend-view-states.md`
- [x] 5.3 Run `./tools/code_quality/check.py check`, `npm run check`, vitest, Playwright, and `openspec validate practice-tasks-and-log --strict`
