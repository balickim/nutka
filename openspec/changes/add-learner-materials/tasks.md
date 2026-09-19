## 1. Backend

- [x] 1.1 Add `internal/materials` with collection names, upload limits, allowed MIME types, body sanitization, and content validation.
- [x] 1.2 Add a migration for the closed `learner_materials` collection with editor, protected file, and autodate fields.
- [x] 1.3 Add `internal/materialsapi` list, create, delete, and file stream routes with cookie identity and assignment ownership checks.
- [x] 1.4 Add unit tests for sanitization and validation, and an HTTP test for scope, file types, and native route closure.

## 2. Frontend

- [x] 2.1 Send `FormData` bodies through the transport boundary and cover it with a transport test.
- [x] 2.2 Add the materials endpoint module, query key, `materialWrite` cache rule, and registry test.
- [x] 2.3 Add `RichTextEditor` and `MaterialList` components.
- [x] 2.4 Add the teacher "Materiały" tab with create form and confirmed deletion.
- [x] 2.5 Add the learner "Materiały od nauczyciela" section to each assignment card.

## 3. Documentation

- [x] 3.1 Add `docs/api/materials.md` and link it from `docs/README.md`.
- [x] 3.2 Add the materials key, the `materialWrite` rule, and multipart transport to the frontend query state constitution.
