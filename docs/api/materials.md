# Learner materials API

The learner materials API stores teacher-authored content for one assignment.
A material contains a title, a rich-text body, and image or PDF attachments.
Only the assigned teacher and the assigned learner can read a material.
The [learner content constitution](../constitutions/learner-content.md) defines the shared scope, storage, and route rules.

## Storage

- The `learner_materials` PocketBase collection stores materials.
- The `assignment` relation links a material to one `teacher_learners` record.
- Deletion of the assignment deletes its materials and their files.
- The `body` field is a PocketBase editor field that holds sanitized HTML.
- The `attachments` field is a protected PocketBase file field.
- The collection has no list, view, create, update, or delete rules.
- Only superusers can use the native record and file routes of the collection.

## Routes

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/assignments/{id}/materials` | Assigned teacher | `{ "items": Material[] }` |
| `GET` | `/api/learners/assignments/{id}/materials` | Assigned learner | `{ "items": Material[] }` |
| `POST` | `/api/teachers/assignments/{id}/materials` | Assigned teacher | `201` with `Material` |
| `DELETE` | `/api/teachers/materials/{id}` | Assigned teacher | `204` |
| `GET` | `/api/teachers/materials/{id}/files/{name}` | Assigned teacher | File stream |
| `GET` | `/api/learners/materials/{id}/files/{name}` | Assigned learner | File stream |

- Each route resolves identity from the session cookie of its realm.
- The `POST` and `DELETE` routes require the `X-Requested-With: fetch` header.
- The list returns materials in reverse order of creation.
- An unrelated account receives `403 unauthorized` without material existence.
- A file name that the material does not hold returns `404 not_found`.

## Create request

The `POST` route accepts `multipart/form-data` with these fields:

| Field | Rule |
| --- | --- |
| `title` | Required. The server trims it. Maximum 200 characters. |
| `body` | Optional HTML. The server sanitizes it before storage. Maximum 200 KB. |
| `attachments` | Optional. Zero to 10 files. Each file is at most 10 MB. |

- A material requires a title and a non-empty body or at least one attachment.
- Attachments accept `image/jpeg`, `image/png`, `image/webp`, `image/gif`, and `application/pdf`.
- A rule violation returns `400 invalid_material`.
- A request body above the route limit returns `413`.

## Body sanitization

- The server keeps `div`, `p`, `br`, `strong`, `b`, `em`, `i`, `u`, `s`, `h3`, `h4`, `ul`, `ol`, `li`, `blockquote`, and `a`.
- A link keeps only an `http`, `https`, or `mailto` `href`.
- The server adds `rel="noreferrer noopener"` and `target="_blank"` to absolute links.
- The server removes every other element and attribute, including scripts, styles, images, and event handlers.
- A body without visible text becomes an empty string.
- Clients render the stored body as HTML without a second sanitizer.

## Material shape

```json
{
  "id": "material-id",
  "assignment": "assignment-id",
  "title": "Gama C-dur",
  "body": "<p>Ćwicz <strong>codziennie</strong>.</p>",
  "attachments": [
    { "name": "nuty_a1b2c3d4e5.pdf", "kind": "pdf", "url": "/api/learners/materials/material-id/files/nuty_a1b2c3d4e5.pdf" }
  ],
  "created_at": "2026-09-19T08:00:00Z"
}
```

- `kind` is `image` or `pdf`.
- `url` is a path in the realm of the caller. Clients prefix it with the configured API base URL.
- `created_at` is a UTC instant.

## File streams

- The file routes stream the stored file through the PocketBase filesystem.
- Images and PDFs use `Content-Disposition: inline`.
- The query parameter `download=1` forces `Content-Disposition: attachment`.
- File responses use `Cache-Control: no-store` so a file does not outlive the session in the browser cache.
