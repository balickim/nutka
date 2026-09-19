# Lesson notes API

The lesson notes API stores one teacher note for one lesson.
A note holds a rich-text body and links to materials of the same assignment.
The [learner content constitution](../constitutions/learner-content.md) defines the shared scope, storage, and route rules.

## Storage

- The `lesson_notes` PocketBase collection stores notes.
- The `lesson` relation links a note to one lesson. A unique index allows one note for each lesson.
- The `assignment` relation copies the lesson assignment when the server creates the note.
- The deletion of the lesson or the assignment deletes the note.
- The `body` field is a PocketBase editor field that holds sanitized HTML.
- The `materials` field links at most 10 `learner_materials` records without cascade delete.

## Routes

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `PUT` | `/api/teachers/lessons/{id}/note` | Assigned teacher | `LessonNote` |
| `DELETE` | `/api/teachers/lessons/{id}/note` | Assigned teacher | `204` |
| `GET` | `/api/teachers/assignments/{id}/lesson-notes` | Assigned teacher | `{ "items": LessonNote[] }` |
| `GET` | `/api/learners/assignments/{id}/lesson-notes` | Assigned learner | `{ "items": LessonNote[] }` |

- The `PUT` and `DELETE` routes require the `X-Requested-With: fetch` header.
- The `PUT` route creates the note of the lesson or replaces its body and materials.
- The `DELETE` route returns `204` also when the lesson has no note.
- The list orders notes by lesson start, newest first.
- An unrelated account receives `403 unauthorized` without lesson or note existence.

## Save request

```json
{ "body": "<p>Refren <strong>wolniej</strong>, tempo 70.</p>", "materials": ["material-id"] }
```

- The server sanitizes the body with the [learner materials](materials.md) allow-list.
- The sanitized body must hold visible text and at most 20 KB. A violation returns `400 invalid_lesson_note`.
- Each material must belong to the lesson assignment. A violation returns `400 invalid_lesson_note`.
- The lesson must have the schedule state `scheduled`. A cancelled lesson returns `409 lesson_cancelled`.
- The lesson start must not be in the future. A future lesson returns `409 lesson_not_started`.

## Note shape

```json
{
  "id": "note-id",
  "lesson": "lesson-id",
  "assignment": "assignment-id",
  "lesson_start_at": "2030-10-09T15:00:00Z",
  "body": "<p>Refren <strong>wolniej</strong>, tempo 70.</p>",
  "materials": [{ "id": "material-id", "title": "Gama C-dur" }],
  "updated_at": "2030-10-09T16:00:00Z"
}
```

- `materials` omits links to deleted materials.
- `lesson_start_at` and `updated_at` are UTC instants.
