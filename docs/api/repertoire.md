# Repertoire API

The repertoire API stores the pieces of one assignment.
A piece has a title, an artist, and a status.
Learner materials that link the same piece are the arrangement versions of that piece.
The [learner content constitution](../constitutions/learner-content.md) defines the shared scope, storage, and route rules.

## Storage

- The `pieces` PocketBase collection stores pieces.
- The `assignment` relation links a piece to one `teacher_learners` record with cascade delete.
- The `status` field is `wish`, `learning`, `playing`, or `repertoire`.
- The server sets `status_changed_at` on create and on each status change.
- The server sets `proposed_by` to `teacher` or `learner` from the route of the create request.
- The `piece` field of `learner_materials` links a material to at most one piece without cascade delete.
- The deletion of a piece clears the `piece` field of its materials and keeps the materials.

## Routes

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/assignments/{id}/pieces` | Assigned teacher | `{ "items": Piece[] }` |
| `GET` | `/api/learners/assignments/{id}/pieces` | Assigned learner | `{ "items": Piece[] }` |
| `POST` | `/api/teachers/assignments/{id}/pieces` | Assigned teacher | `201` with `Piece` |
| `POST` | `/api/learners/assignments/{id}/pieces` | Assigned learner | `201` with `Piece` |
| `PATCH` | `/api/teachers/pieces/{id}` | Assigned teacher | `Piece` |
| `DELETE` | `/api/teachers/pieces/{id}` | Assigned teacher | `204` |
| `DELETE` | `/api/learners/pieces/{id}` | Assigned learner | `204` |

- The `POST`, `PATCH`, and `DELETE` routes require the `X-Requested-With: fetch` header.
- The list orders pieces by `status_changed_at`, newest first.
- An unrelated account receives `403 unauthorized` without piece existence.

## Teacher writes

```json
{ "title": "Hallelujah", "artist": "Leonard Cohen", "status": "learning" }
```

- The server trims `title` and `artist`.
- `title` is required and holds at most 200 characters. `artist` holds at most 200 characters.
- The create route uses the status `learning` when the request has no status.
- The `PATCH` route changes only the fields in the request.
- The status order is free. A piece can go back from `repertoire` to `learning`.
- A rule violation returns `400 invalid_piece`.

## Learner writes

```json
{ "title": "Perfect", "artist": "Ed Sheeran" }
```

- The learner create route always stores the status `wish`. It ignores a `status` field.
- The learner delete route deletes only a piece with `proposed_by` `learner` and the status `wish`. Another piece returns `409 piece_locked`.
- A learner write on an inactive assignment returns `409 assignment_inactive`.

## Piece shape

```json
{
  "id": "piece-id",
  "assignment": "assignment-id",
  "title": "Hallelujah",
  "artist": "Leonard Cohen",
  "status": "learning",
  "status_changed_at": "2030-10-01T10:00:00Z",
  "proposed_by": "learner",
  "material_count": 2,
  "latest_material_at": "2030-10-08T10:00:00Z",
  "created_at": "2030-09-20T10:00:00Z"
}
```

- `material_count` counts the materials that link the piece.
- `latest_material_at` is the creation instant of the newest linked material, or `null`.
- Every instant is UTC.

## Arrangement versions

- The versions of a piece are its linked materials in order of creation.
- A client labels the oldest material "wersja 1" and shows the newest version first.
