## Purpose

Defines teacher-authored teaching materials that belong to one teacher-learner assignment.

## ADDED Requirements

### Requirement: Teacher adds materials for one learner
The assigned teacher SHALL add a material with a title and a rich-text body, image attachments, PDF attachments, or a combination of them. A material SHALL belong to exactly one assignment.

#### Scenario: Teacher adds a material with files
- **WHEN** the assigned teacher submits a title, a formatted body, one image, and one PDF
- **THEN** the system stores the material for that assignment and returns both attachments

#### Scenario: Unrelated teacher adds a material
- **WHEN** a teacher who does not own the assignment submits a material
- **THEN** the system rejects the request with `unauthorized` and stores nothing

#### Scenario: Teacher submits an empty material
- **WHEN** the teacher submits a title without body text and without attachments
- **THEN** the system rejects the request with `invalid_material`

#### Scenario: Teacher attaches a disallowed file
- **WHEN** the teacher attaches a file that is not a JPEG, PNG, WebP, GIF, or PDF, or exceeds 10 MB
- **THEN** the system rejects the request with `invalid_material`

### Requirement: Only the assignment pair reads materials
The system SHALL return materials and attachment files only to the assigned teacher and the assigned learner.

#### Scenario: Learner reads own materials
- **WHEN** the assigned learner opens the dashboard
- **THEN** the assignment card lists its materials newest first with rendered text, images, and PDF links

#### Scenario: Another learner requests a material file
- **WHEN** a learner who does not own the assignment requests an attachment URL
- **THEN** the system responds with `unauthorized`

#### Scenario: Native collection routes
- **WHEN** a persona session calls the native record or file route of `learner_materials`
- **THEN** the system returns no material data

### Requirement: Rich text cannot run code
The system SHALL sanitize the body on the server before storage and keep only the formatting the editor produces.

#### Scenario: Body contains a script
- **WHEN** the teacher submits a body with a script, an event handler, or a `javascript:` link
- **THEN** the stored body keeps the text and formatting and drops the active content

### Requirement: Teacher deletes a material
The assigned teacher SHALL delete a material after a confirmation that names the effect on the learner view.

#### Scenario: Teacher confirms deletion
- **WHEN** the teacher confirms deletion of a material
- **THEN** the system deletes the record and its files and the material disappears from both views
