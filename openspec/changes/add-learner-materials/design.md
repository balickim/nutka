## Context

Nutka keeps every PocketBase collection closed to persona sessions. Custom `/api/teachers/*` and `/api/learners/*` routes resolve identity from HttpOnly session cookies. See `proposal.md` for the motivation and `specs/learner-materials/spec.md` for required behavior.

PocketBase provides an editor field for HTML, a file field with size, count, and MIME limits, protected file storage, and a filesystem abstraction for local disk or S3.

## Goals / Non-Goals

**Goals:**

- Let the assigned teacher add rich text, images, and PDFs for one learner.
- Let only the assigned learner and teacher read them.
- Use PocketBase field validation and file storage without custom storage code.

**Non-Goals:**

- Edit an existing material. The teacher deletes and adds it again.
- Share one material with many learners.
- Generate image thumbnails, preview PDFs in the page, or track learner reads.
- Add a third-party rich-text editor to the frontend.

## Decisions

### Scope a material to an assignment

The `assignment` relation points to `teacher_learners` with cascade delete. The assignment already encodes the teacher-learner pair, so one ownership check covers both personas. A learner with two teachers sees each teacher's materials on that teacher's card.

### Keep the collection closed and stream files through persona routes

The collection has no API rules. Native PocketBase file URLs need a short-lived file token that the cookie session cannot issue to the browser. The persona file route checks the cookie and the assignment owner, then calls `filesystem.System.Serve`. PocketBase then handles range requests, inline disposition for images and PDFs, and its content security headers.

Alternative: open `viewRule` and issue PocketBase file tokens. Rejected because it exposes native record routes and requires token refresh before every file open.

### Sanitize HTML on the server

The editor field does not sanitize HTML. The backend applies one bluemonday allow-list before storage, so every client can render the stored body directly. The allow-list matches the toolbar: paragraphs, headings, emphasis, lists, quotes, and safe links.

### Use a dependency-free contentEditable editor

The toolbar uses `document.execCommand` for bold, italic, underline, headings, paragraphs, and lists. The API is deprecated but supported by current browsers. Server sanitization removes pasted markup outside the allow-list.

## Risks / Trade-offs

- `execCommand` may lose browser support → Replace the editor component. The API contract stays the same.
- A 32 MB global body limit blocks large uploads → The create route binds its own limit of ten 10 MB files plus the body.
- Stored names carry a PocketBase suffix → The client removes the suffix for display only.
