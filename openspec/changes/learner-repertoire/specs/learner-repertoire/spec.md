## Purpose

Lets a learner see the pieces they learn, play, and keep in repertoire, ask for pieces they want to play, and find the arrangement versions of each piece.

## ADDED Requirements

### Requirement: Teacher manages pieces

The assigned teacher SHALL create, edit, change the status of, and delete pieces of an assignment. A piece SHALL have a title of 1 to 200 characters, an artist of at most 200 characters, and a status of `wish`, `learning`, `playing`, or `repertoire`. The server SHALL set `status_changed_at` on create and on each status change.

#### Scenario: Teacher adds a piece

- **WHEN** the teacher creates a piece with a title and the status `learning`
- **THEN** the API returns `201` with `proposed_by` `teacher` and the current `status_changed_at`

#### Scenario: Teacher changes the status

- **WHEN** the teacher changes the status of a piece from `learning` to `playing`
- **THEN** the API stores the new status and a new `status_changed_at`

#### Scenario: Status goes back

- **WHEN** the teacher changes the status of a piece from `repertoire` to `learning`
- **THEN** the API accepts the change

#### Scenario: Invalid piece

- **WHEN** the teacher sends an empty title or an unknown status
- **THEN** the API returns `400 invalid_piece` and stores nothing

### Requirement: Learner adds wishes

The assigned learner SHALL add a piece to an active assignment. The server SHALL store the status `wish` and `proposed_by` `learner`. The learner SHALL delete only their own piece while its status is `wish`.

#### Scenario: Learner adds a wish

- **WHEN** the learner adds a piece with a status `playing` in the request
- **THEN** the API stores the piece with the status `wish`

#### Scenario: Learner deletes a teacher piece

- **WHEN** the learner deletes a piece that the teacher created
- **THEN** the API returns `409 piece_locked` and keeps the piece

#### Scenario: Learner deletes an accepted wish

- **WHEN** the learner deletes their piece after the teacher changed its status to `learning`
- **THEN** the API returns `409 piece_locked` and keeps the piece

#### Scenario: Inactive assignment

- **WHEN** the learner adds a piece to an inactive assignment
- **THEN** the API returns `409 assignment_inactive` and stores nothing

### Requirement: Pieces are assignment scoped

Only the assigned teacher and the assigned learner SHALL read the pieces of an assignment. Each piece in the list SHALL carry `material_count` and `latest_material_at`.

#### Scenario: Unrelated teacher

- **WHEN** another teacher changes a piece
- **THEN** the API returns `403 unauthorized` without piece existence

#### Scenario: Learner calls a teacher route

- **WHEN** a learner session calls a teacher piece route
- **THEN** the API returns `403 unauthorized` and stores nothing

### Requirement: Materials link a piece

A teacher material SHALL link at most one piece of the same assignment. The teacher SHALL set the link on create and change it later. Deleting a piece SHALL clear the link and keep the material.

#### Scenario: Piece of another assignment

- **WHEN** the teacher creates a material with a piece of another assignment
- **THEN** the API returns `400 invalid_material` and stores nothing

#### Scenario: Piece deleted

- **WHEN** the teacher deletes a piece with two linked materials
- **THEN** both materials stay and have no piece

### Requirement: Learner sees the repertoire

The learner Utwory screen SHALL group pieces into "Uczę się", "Gram", and "W repertuarze". It SHALL show a "Chcę zagrać" section with the wishes and a form to add one. Each piece SHALL list its versions newest first with the labels "wersja 1", "wersja 2", and so on, by creation order. Materials without a piece SHALL show under "Inne materiały".

#### Scenario: Two versions

- **WHEN** a piece has two linked materials
- **THEN** the piece shows the newer material first with the label "wersja 2"

### Requirement: Teacher sees wishes first

The teacher learner view SHALL have an Utwory tab with the learner wishes first, a form to add a piece, and a status control on each piece. The teacher materials tab SHALL offer a piece choice on create and on each material.

#### Scenario: Teacher accepts a wish

- **WHEN** the teacher starts a wish from the Utwory tab
- **THEN** the piece moves to the status `learning` and shows under "Uczę się" for the learner
