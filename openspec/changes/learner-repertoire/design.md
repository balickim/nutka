## Context

See `proposal.md` for motivation and `specs/learner-repertoire/spec.md` for behavior. `docs/constitutions/learner-content.md` defines the shared rules for assignment-scoped learner content.

- `internal/personaroute` resolves the caller and assignment ownership.
- `learner_materials` holds teacher materials with a rich-text body and files.

## Goals / Non-Goals

**Goals:**

- One piece record per piece of one assignment, with a status that the teacher changes in one step.
- Arrangement versions without a new entity: materials that link the same piece.
- Learner wishes that the teacher sees first.

**Non-Goals:**

- A shared piece catalog across learners.
- A forced status order. A piece can go back to `learning`.
- Business events for pieces.

## Decisions

### Collection `pieces`

Fields: `assignment` (cascade), `title` (required, max 200), `artist` (max 200), `status` (select), `status_changed_at` (date, set by the server), `proposed_by` (`teacher` or `learner`, set by the server), and autodates. The server sets `status_changed_at` on create and on each status change.

### Material link

`learner_materials.piece` is an optional single relation to `pieces` without cascade. PocketBase clears the link when the piece is deleted, so the material stays and moves to other materials. The API rejects a piece of another assignment with `400 invalid_material`. `PATCH /api/teachers/materials/{id}` changes only `piece`.

### Pure rules in `internal/repertoire`

`Normalize` trims the title and artist. `Validate` checks the lengths and the status. `LearnerMayDelete(proposedBy, status)` allows the delete only for a learner piece in `wish`. The API derives `proposed_by` from the route, so a request cannot claim another author.

### Learner status is fixed

The learner create route has no status field. The server always stores `wish`, so a learner cannot set another status.

### Inactive assignment

`personaroute.ErrInactive` maps to `409 assignment_inactive`. The learner create and delete routes check `active` on the assignment. Teacher routes stay open on an inactive assignment, like materials. Later learner write routes reuse the same error.

### List counters

The piece list returns `material_count` and `latest_material_at` for each piece. The API computes them from one material query of the assignment. The list is ordered by `status_changed_at`, newest first.

### Client grouping

A pure `groupRepertoire(pieces, materials)` returns wishes, the three status groups, versions per piece, and materials without a piece. Versions are ordered by `created_at`. The label "wersja N" uses the creation order.

### Cache rules

`pieceWrite(assignmentId)` invalidates the pieces and materials of one assignment for both personas, because a delete clears material links. `materialWrite(assignmentId)` also invalidates the pieces of that assignment, because the counters change.

## Risks / Trade-offs

- [The teacher deletes a piece with versions] → The materials stay under other materials. The confirmation says so.
- [Two teachers of one learner] → Each assignment has its own pieces, like other learner content.
