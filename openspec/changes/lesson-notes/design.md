## Context

See `proposal.md` for motivation and `specs/lesson-notes/spec.md` for behavior. `docs/constitutions/learner-content.md` defines the shared rules for assignment-scoped learner content.

- `materials.SanitizeBody` is the one allow-list for teacher rich text.
- `internal/personaroute` resolves the caller and assignment ownership.
- The teacher calendar returns every lesson of the teacher without a lower date bound. The learner calendar returns only future lessons.

## Goals / Non-Goals

**Goals:**

- One note per lesson with an upsert route, so the client never tracks note identifiers for writes.
- Reuse the material sanitizer and the persona route rule.

**Non-Goals:**

- Drafts, publish state, or read receipts. A saved note is visible at once.
- Attachments on notes. A note links existing materials instead.
- Business events for notes.

## Decisions

### Upsert by lesson

`PUT /api/teachers/lessons/{id}/note` creates or replaces the note of the lesson. A unique index on `lesson` enforces one note. The route derives `assignment` from the lesson, so a request cannot move a note to another assignment.

### Eligibility in a pure package

`internal/lessonnotes.CheckLesson(startAt, scheduleState, now)` returns `ErrNotStarted` or `ErrCancelled`. `Validate(body, materialCount)` checks the sanitized body and the material limit. The API checks material ownership against the lesson assignment. The API takes a clock, so tests control the start rule.

### Material links

`materials` is a multi relation to `learner_materials` without cascade. The response resolves titles and skips missing materials, so a deleted material disappears from the note without a write.

### Teacher past lessons from the calendar

The teacher Notatki tab reads past lessons from the teacher calendar that the shell already loads. No new lesson read is needed.

### Editor initial value

`RichTextEditor` gains an optional `initialHtml`. The editor writes it into the editable area once on mount. The value comes from the server and is already sanitized.

### Cache rule

`noteWrite(assignmentId)` invalidates the note lists of both personas for that assignment only, like `materialWrite`.

## Risks / Trade-offs

- [A lesson is cancelled after the teacher writes a note] → The note stays and remains visible. The teacher can delete it.
- [The learner calendar has no past lessons] → The learner screens read lesson dates from the note list, which carries `lesson_start_at`.
