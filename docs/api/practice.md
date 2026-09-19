# Practice API

The practice API stores the practice tasks of one assignment and the practice sessions of the learner.
It also returns practice summaries since the previous lesson.
The [learner content constitution](../constitutions/learner-content.md) defines the shared scope, storage, and route rules.

## Storage

- The `practice_tasks` PocketBase collection stores tasks.
- A task has `assignment`, optional `lesson`, `title`, `details`, optional `suggested_minutes`, optional `piece` and `material`, `status`, and `position`.
- The task status is `active`, `done`, or `archived`. The active tasks are the plan for the week.
- The `lesson`, `piece`, and `material` relations have no cascade delete. Deleting the target clears the link and keeps the task.
- The `practice_sessions` PocketBase collection stores sessions.
- A session has `assignment`, `practiced_on`, optional `minutes`, `tasks`, and `comment`.
- `practiced_on` is a local date `YYYY-MM-DD` in the teacher timezone. A practice session belongs to a day, not to an instant.
- The deletion of the assignment deletes its tasks and sessions.

## Routes

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/assignments/{id}/practice-tasks` | Assigned teacher | `{ "items": PracticeTask[] }` |
| `GET` | `/api/learners/assignments/{id}/practice-tasks` | Assigned learner | `{ "items": PracticeTask[] }` |
| `PUT` | `/api/teachers/assignments/{id}/practice-plan` | Assigned teacher | `{ "items": PracticeTask[] }` |
| `PATCH` | `/api/teachers/practice-tasks/{id}` | Assigned teacher | `PracticeTask` |
| `POST` | `/api/learners/assignments/{id}/practice-sessions` | Assigned learner | `201` with `PracticeSession` |
| `DELETE` | `/api/learners/practice-sessions/{id}` | Assigned learner | `204` |
| `GET` | `/api/teachers/assignments/{id}/practice-sessions` | Assigned teacher | `SessionPage` |
| `GET` | `/api/learners/assignments/{id}/practice-sessions` | Assigned learner | `SessionPage` |
| `GET` | `/api/teachers/assignments/{id}/practice-summary` | Assigned teacher | `AssignmentSummary` |
| `GET` | `/api/learners/assignments/{id}/practice-summary` | Assigned learner | `AssignmentSummary` |
| `GET` | `/api/teachers/practice-summaries?date=YYYY-MM-DD` | Teacher | `{ "items": LessonSummary[] }` |

- The `PUT`, `PATCH`, `POST`, and `DELETE` routes require the `X-Requested-With: fetch` header.
- An unrelated account receives `403 unauthorized` without task or session existence.
- The task list takes `status` with the value `active`, `done`, `archived`, or `all`. The default is `active`.
- The task list orders active tasks by `position`. It orders done and archived tasks by the latest change, newest first.
- The session list takes `page` from 1. A page holds 20 sessions ordered by `practiced_on`, newest first.

## Plan write

```json
{
  "lesson": "lesson-id",
  "keep": ["task-id"],
  "done": ["task-id"],
  "archive": ["task-id"],
  "create": [{ "title": "Refren, tempo 70", "details": "", "suggested_minutes": 10, "piece": "piece-id", "material": "" }]
}
```

- The server writes the whole plan in one transaction.
- Each named task must belong to the assignment and appear in one list only.
- `lesson`, `piece`, and `material` must belong to the assignment. An empty value means no link.
- A new task needs a title of 1 to 200 characters. `details` holds at most 1000 characters of plain text. `suggested_minutes` is `null` or 1 to 120.
- The plan holds at most 50 named and new tasks.
- A rule violation returns `400 invalid_practice_plan` and changes nothing.
- The new positions follow `keep`, then the other active tasks in stored order, then `create`.
- The response lists the active tasks.

## Task update

- The `PATCH` route changes `title`, `details`, `suggested_minutes`, and `status`. It keeps the other fields.
- A rule violation returns `400 invalid_practice_task`.

## Session write

```json
{ "practiced_on": "2030-10-09", "minutes": 20, "tasks": ["task-id"], "comment": "Takt 5 jeszcze nie wychodzi." }
```

- The server reads today from its clock and the teacher timezone.
- `practiced_on` is today or a day at most 14 days before today.
- `minutes` is `null` or 1 to 240. `comment` holds at most 500 characters after trimming.
- Each task must belong to the assignment. The server removes repeated tasks.
- A rule violation returns `400 invalid_practice_session`.
- A write on an inactive assignment returns `409 assignment_inactive`.
- The learner deletes a session at most 7 days after its creation. A later delete returns `409 practice_session_locked`.

## Summary window

- The window starts on the local day of the latest scheduled lesson of the assignment whose start is before the reference instant.
- The window includes the lesson day, because a learner often practices after the lesson.
- Without such a lesson, the window starts 6 days before today and covers 7 days.
- The assignment summary uses the current instant as the reference.
- The teacher day summaries use the start of each lesson on the date as the reference, so each window starts at the previous lesson.

## Shapes

```json
{
  "id": "task-id",
  "assignment": "assignment-id",
  "lesson": "lesson-id",
  "title": "Refren, tempo 70",
  "details": "",
  "suggested_minutes": 10,
  "piece": { "id": "piece-id", "title": "Hallelujah" },
  "material": null,
  "status": "active",
  "position": 1,
  "created_at": "2030-10-09T12:00:00Z"
}
```

```json
{
  "id": "session-id",
  "practiced_on": "2030-10-09",
  "minutes": 20,
  "tasks": [{ "id": "task-id", "title": "Refren, tempo 70" }],
  "comment": "Takt 5 jeszcze nie wychodzi.",
  "created_at": "2030-10-09T18:00:00Z",
  "deletable": true
}
```

```json
{
  "since_on": "2030-10-02",
  "days": 4,
  "minutes": 65,
  "sessions": 5,
  "tasks": [{ "task": "task-id", "title": "Refren, tempo 70", "count": 3 }],
  "comments": [{ "practiced_on": "2030-10-08", "comment": "Takt 5 jeszcze nie wychodzi." }],
  "today": "2030-10-09",
  "recent_days": ["2030-10-03", "2030-10-08"]
}
```

- `SessionPage` is `{ "items": PracticeSession[], "page": 1, "per_page": 20, "total": 5 }`.
- `deletable` is `true` only for the learner within the delete window.
- `days` counts local days with at least one session. `minutes` adds the stated minutes.
- `tasks` omits deleted tasks and orders the rest by count, highest first.
- `comments` are newest first.
- Only the assignment summary has `today` and `recent_days`. `recent_days` lists the practice days of the 28 days that end today.
- `LessonSummary` is `{ "lesson": "lesson-id", "assignment": "assignment-id", "summary": Summary }`.
