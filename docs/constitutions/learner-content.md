# Learner content constitution

This document defines the rules that every assignment-scoped learner content type follows.
Learner content is teaching content that a teacher and a learner share inside one assignment.
The [learner materials API](../api/materials.md) and the [lesson notes API](../api/lesson-notes.md) define the current learner content types.

## Scope

- Each learner content record belongs to exactly one `teacher_learners` assignment.
- Only the assigned teacher and the assigned learner can read a learner content record.
- A learner with two teachers sees the content of each assignment separately.
- The deletion of an assignment deletes its learner content and the files of that content.

## Storage

- Each learner content type uses its own PocketBase collection.
- The `assignment` relation of the collection uses cascade delete.
- The collection has no list, view, create, update, or delete rules.
- Only superusers can use the native record and file routes of the collection.
- File fields are protected. The persona file routes stream the files.
- The collection stores datetimes as UTC instants.

## Routes

- Teacher routes use `/api/teachers/*`. Learner routes use `/api/learners/*`.
- The `internal/personaroute` package resolves the caller from the session cookie of the route realm.
- The `internal/personaroute` package checks that the caller owns the assignment.
- An unrelated account receives `403 unauthorized` and learns nothing about the existence of the record.
- A write route requires the `X-Requested-With: fetch` header.
- A request field never replaces the caller identity or the author role.

## History

- A learner content write does not append a business event.
- The business history stays limited to plans, lessons, and money, as the [commercial policy decision](../adr/0001-commercial-policy-and-history.md) defines.

## Presentation

- A client reads the local day and week of learner content in the teacher timezone.
- The learner panel shows learner content on the learner screen that owns the content type.
