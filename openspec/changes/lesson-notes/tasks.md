## 1. Backend

- [x] 1.1 Add `internal/lessonnotes` with names, limits, `CheckLesson`, and `Validate`, and table tests
- [x] 1.2 Add the `lesson_notes` migration with unique `lesson`, cascade relations, editor body, and material relation
- [x] 1.3 Add `internal/lessonnotesapi` with `PUT` and `DELETE` note routes and teacher and learner list routes, and register it with the clock in `main.go`
- [x] 1.4 Add HTTP tests: future and cancelled lessons, foreign material, upsert keeps one note, learner write rejected, foreign teacher rejected, learner list, delete
- [x] 1.5 Run `go build ./...` and `go test ./...`

## 2. Frontend data

- [x] 2.1 Add `api/lesson-notes.ts` with types and requests
- [x] 2.2 Add query keys, queries, the `noteWrite` rule, and a `keys.test.ts` case
- [x] 2.3 Add `initialHtml` to `RichTextEditor`

## 3. Teacher screens

- [x] 3.1 Add a note editor component with body and material checkboxes
- [x] 3.2 Open the editor from the Today card after the outcome `completed`
- [x] 3.3 Add the Notatki tab with past lessons, notes, and add, edit, and delete controls

## 4. Learner screens

- [x] 4.1 Show the latest note on the Start screen
- [x] 4.2 List notes on the lessons screen

## 5. Tests, docs, and verification

- [x] 5.1 Add a Playwright flow: the learner sees the latest note on Start and the note list on lessons
- [x] 5.2 Add `docs/api/lesson-notes.md` and update `docs/README.md`, `learner-content.md`, and `frontend-view-states.md`
- [x] 5.3 Run `./tools/code_quality/check.py check`, `npm run check`, vitest, Playwright, and `openspec validate lesson-notes --strict`
