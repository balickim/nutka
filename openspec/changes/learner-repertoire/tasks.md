## 1. Backend

- [x] 1.1 Add `internal/repertoire` with names, statuses, `Normalize`, `Validate`, and `LearnerMayDelete`, and table tests
- [x] 1.2 Add `personaroute.ErrInactive` with `409 assignment_inactive` and `RequireActive`
- [x] 1.3 Add the migration that creates `pieces` and adds `learner_materials.piece`
- [x] 1.4 Add `internal/repertoireapi` with teacher and learner list, create, update, and delete routes, and register it in `main.go`
- [x] 1.5 Add the `piece` field to material create, the material DTO, and `PATCH /api/teachers/materials/{id}`
- [x] 1.6 Add HTTP tests: learner status forced to `wish`, learner delete of a teacher piece and of an accepted wish, inactive assignment, foreign teacher, foreign piece on a material, piece delete clears links, counters
- [x] 1.7 Run `go build ./...` and `go test ./...`

## 2. Frontend data

- [x] 2.1 Add `api/pieces.ts` and the material `piece` field and pin request
- [x] 2.2 Add piece query keys, queries, the `pieceWrite` rule, the `materialWrite` extension, and `keys.test.ts` cases
- [x] 2.3 Add pure `groupRepertoire` with tests

## 3. Screens

- [x] 3.1 Replace the learner Utwory screen with status groups, the wish form, versions, and other materials
- [x] 3.2 Add the teacher Utwory tab with wishes first, a create form, status changes, and delete
- [x] 3.3 Add the piece choice to the teacher material form and to each material

## 4. Tests, docs, and verification

- [x] 4.1 Add a Playwright flow: the learner adds a wish, the teacher starts it, and the learner sees it under "Uczę się"
- [x] 4.2 Add `docs/api/repertoire.md` and update `docs/api/materials.md`, `docs/README.md`, `learner-content.md`, and `frontend-view-states.md`
- [x] 4.3 Run `./tools/code_quality/check.py check`, `npm run check`, vitest, Playwright, and `openspec validate learner-repertoire --strict`
