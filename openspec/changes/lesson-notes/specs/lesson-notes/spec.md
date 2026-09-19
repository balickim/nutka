## Purpose

Lets a teacher leave a short note after each lesson and lets the learner read what went well and what to work on next.

## ADDED Requirements

### Requirement: Teacher writes one note per lesson

The assigned teacher SHALL create or replace one note for a lesson whose start is not in the future and whose schedule state is `scheduled`. The note SHALL hold a sanitized body with visible text of at most 20 KB. The note MAY link at most 10 materials of the same assignment.

#### Scenario: Teacher saves a note after the lesson

- **WHEN** the teacher saves a note for a lesson that started one hour ago
- **THEN** the API stores one note for that lesson and returns it with the lesson start

#### Scenario: Teacher replaces the note

- **WHEN** the teacher saves a second note for the same lesson
- **THEN** the API replaces the body and materials of the existing note and keeps one note

#### Scenario: Future lesson

- **WHEN** the teacher saves a note for a lesson that starts tomorrow
- **THEN** the API returns `409 lesson_not_started` and stores nothing

#### Scenario: Cancelled lesson

- **WHEN** the teacher saves a note for a cancelled lesson
- **THEN** the API returns `409 lesson_cancelled` and stores nothing

#### Scenario: Material of another assignment

- **WHEN** the teacher links a material of another assignment
- **THEN** the API returns `400 invalid_lesson_note` and stores nothing

### Requirement: Teacher deletes a note

The assigned teacher SHALL delete the note of a lesson. The learner SHALL no longer see it.

#### Scenario: Teacher deletes

- **WHEN** the teacher deletes the note of a lesson
- **THEN** the API returns `204` and the learner list no longer contains the note

### Requirement: Notes are assignment scoped

Only the assigned teacher and the assigned learner SHALL read the notes of an assignment. Only the assigned teacher SHALL write them. The list SHALL order notes by lesson start, newest first.

#### Scenario: Learner reads notes

- **WHEN** the assigned learner reads the notes of the assignment
- **THEN** the API returns the notes with body, lesson start, and linked material titles

#### Scenario: Learner tries to write

- **WHEN** a learner session calls the teacher note route
- **THEN** the API returns `403 unauthorized` and stores nothing

#### Scenario: Unrelated teacher

- **WHEN** another teacher saves a note for the lesson
- **THEN** the API returns `403 unauthorized` without lesson existence

### Requirement: Teacher adds a note after the lesson

The teacher Today card SHALL offer a note editor for a lesson after the teacher records the outcome `completed`. The teacher learner view SHALL list the past lessons of the assignment with their notes and SHALL offer add, edit, and delete controls.

#### Scenario: Teacher marks a lesson completed

- **WHEN** the teacher records `completed` on a Today card
- **THEN** the card shows a control that opens the note editor for that lesson

### Requirement: Learner sees notes

The learner Start screen SHALL show the latest note of the selected assignment with its lesson date. The learner lessons screen SHALL list every note of the selected assignment with its lesson date and linked material titles.

#### Scenario: No notes yet

- **WHEN** the selected assignment has no notes
- **THEN** the Start screen shows no note block and the lessons screen shows one sentence that notes appear after lessons
