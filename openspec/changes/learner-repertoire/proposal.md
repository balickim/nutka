## Why

The landing page promises that the learner chooses the pieces to play. Today the learner sees a flat list of materials and cannot tell which piece they learn, which pieces they already play, or which arrangement is newer. The teacher has no place to track the repertoire of a learner or to see the pieces that the learner wants to play.

## What Changes

- Add pieces to an assignment with a title, an optional artist, and a status: `wish`, `learning`, `playing`, or `repertoire`.
- Let the teacher create, edit, change the status of, and delete pieces.
- Let the learner add a piece as a wish and delete their own piece while it is a wish.
- Reject a learner write on an inactive assignment with `409 assignment_inactive`.
- Add an optional `piece` link to materials. The materials of one piece, ordered by creation, are the arrangement versions of that piece.
- Replace the learner Utwory screen with status groups, a wish form, versions per piece, and other materials.
- Add a teacher Utwory tab and a piece choice to the teacher materials tab.

## Capabilities

### New Capabilities

- `learner-repertoire`: Assignment-scoped pieces, their statuses, learner wishes, and arrangement versions.

### Modified Capabilities

None.

## Impact

- Backend: new `pieces` collection, new `learner_materials.piece` field, `internal/repertoire`, `internal/repertoireapi`, `materialsapi` piece field and pin route, and `personaroute.ErrInactive`.
- Frontend: `api/pieces.ts`, piece queries and cache rules, the learner Utwory screen, the teacher Utwory tab, and the teacher materials tab.
- Docs: `docs/api/repertoire.md`, `docs/api/materials.md`, `docs/README.md`, `docs/constitutions/learner-content.md`, `docs/constitutions/frontend-view-states.md`.
