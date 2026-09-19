## Why

After a lesson, the learner has no record of what went well and what to work on. Adult learners want to understand what they practice, and the landing page promises a plan on the learner profile. The teacher has no place for a short note tied to one lesson. Materials are general content and do not say which lesson they came from.

## What Changes

- Add one teacher-authored note per started, not cancelled lesson, with formatted text and links to materials of the same assignment.
- Let the teacher create, replace, and delete the note. The learner sees it immediately after save and cannot change it.
- Add an "after the lesson" editor to the teacher Today card after the teacher records the outcome.
- Add a "Notatki" tab to the teacher learner view with past lessons and their notes.
- Show the latest note on the learner Start screen and all notes on the learner lessons screen.

## Capabilities

### New Capabilities

- `lesson-notes`: Lesson-scoped teacher notes, their eligibility, access, and presentation.

### Modified Capabilities

None.

## Impact

- Backend: new `lesson_notes` collection, `internal/lessonnotes`, and `internal/lessonnotesapi`.
- Frontend: `api/lesson-notes.ts`, query keys and cache rule, `RichTextEditor` initial value, teacher Today card, teacher Notatki tab, learner Start and lessons screens.
- Docs: `docs/api/lesson-notes.md`, `docs/README.md`, `docs/constitutions/learner-content.md`, `docs/constitutions/frontend-view-states.md`.
