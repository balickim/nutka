## Why

Teachers need to give each learner individual teaching content: notes, sheet-music photos, and PDF exercises. Nutka has no place for such content. Learners receive it today through external channels.

## What Changes

- Add a closed `learner_materials` PocketBase collection scoped to one teacher-learner assignment.
- Store the body in a PocketBase editor field and attachments in a protected PocketBase file field.
- Add teacher routes to list, create, and delete materials, and a learner route to list them.
- Stream attachments through persona routes that check the session cookie and the assignment owner.
- Sanitize the rich-text body on the server before storage.
- Add a "Materiały" tab to the teacher student view with a rich-text editor and file picker.
- Add a "Materiały od nauczyciela" section to each learner assignment card.

## Capabilities

### New Capabilities

- `learner-materials`: Defines teacher-authored, assignment-scoped materials and their access rules.

### Modified Capabilities

None.

## Impact

- Adds a migration, `internal/materials`, and `internal/materialsapi` to `apps/backend`.
- Adds the `github.com/microcosm-cc/bluemonday` HTML sanitizer to the backend.
- Adds multipart support to the frontend transport boundary.
- Adds one query key and one cache rule to the frontend registry.
- Does not change scheduling, commercial, or authentication contracts.
