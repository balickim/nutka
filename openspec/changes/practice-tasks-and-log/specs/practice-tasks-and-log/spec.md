## Purpose

Lets a teacher set practice tasks after a lesson, lets a learner record practice without scoring, and shows the teacher what happened since the last lesson.

## ADDED Requirements

### Requirement: Teacher sets the practice plan

The assigned teacher SHALL keep, complete, archive, and create practice tasks of an assignment in one atomic write. A task SHALL have a title of 1 to 200 characters, details of at most 1000 characters, and optional suggested minutes from 1 to 120. A linked piece or material SHALL belong to the same assignment.

#### Scenario: Teacher saves a plan after a lesson

- **WHEN** the teacher completes one active task and creates two tasks for a lesson
- **THEN** the API returns the active tasks in the new order and the new tasks link the lesson

#### Scenario: Foreign task in the plan

- **WHEN** the plan names a task of another assignment
- **THEN** the API returns `400 invalid_practice_plan` and changes nothing

#### Scenario: Invalid new task

- **WHEN** one of the new tasks has an empty title
- **THEN** the API returns `400 invalid_practice_plan` and changes no task

### Requirement: Learner records practice

The assigned learner SHALL record a practice session on an active assignment for a local day from today back to 14 days before today in the teacher timezone. A session SHALL have optional minutes from 1 to 240, tasks of the same assignment, and a comment of at most 500 characters. The learner SHALL delete an own session within 7 days after it was created.

#### Scenario: Learner records today

- **WHEN** the learner records a session for today with two tasks and a comment
- **THEN** the API returns `201` with the session

#### Scenario: Future day

- **WHEN** the learner records a session for tomorrow
- **THEN** the API returns `400 invalid_practice_session`

#### Scenario: Too old

- **WHEN** the learner records a session for 15 days ago
- **THEN** the API returns `400 invalid_practice_session`

#### Scenario: Foreign task

- **WHEN** the session names a task of another assignment
- **THEN** the API returns `400 invalid_practice_session`

#### Scenario: Late delete

- **WHEN** the learner deletes a session created 8 days ago
- **THEN** the API returns `409 practice_session_locked`

#### Scenario: Inactive assignment

- **WHEN** the learner records a session on an inactive assignment
- **THEN** the API returns `409 assignment_inactive`

### Requirement: Practice summary

The API SHALL return, for an assignment, the practice days, the minutes, the sessions, the session count of each task, and the comments since the start of the summary window. The window SHALL start on the local day of the latest started lesson, or 6 days before today without such a lesson. The teacher SHALL read the summaries for every lesson of one local day in one request.

#### Scenario: Window after a lesson

- **WHEN** the last lesson was 3 days ago and the learner practiced 2 days ago and 5 days ago
- **THEN** the summary counts one practice day

#### Scenario: Warsaw day boundary

- **WHEN** the learner records a session at 00:30 Warsaw time
- **THEN** the session belongs to that Warsaw day

### Requirement: Learner practice screen

The learner SHALL see the active tasks with details, suggested minutes, and linked piece and material on `/learners/practice` and on Start. One button SHALL open the session form. The screen SHALL show the practice days of the last 4 weeks without a score.

#### Scenario: No tasks

- **WHEN** the assignment has no active tasks
- **THEN** the screen says that the teacher adds tasks after a lesson and still offers the session form

### Requirement: Teacher practice views

The teacher after-lesson flow SHALL show the active tasks with keep, done, and archive choices and a form for new tasks. The Today card SHALL show the practice summary since the previous lesson with the learner comments. The teacher learner view SHALL have a Ćwiczenia tab with tasks and sessions.

#### Scenario: Summary on Today

- **WHEN** the learner practiced on 4 days for 65 minutes since the previous lesson
- **THEN** the Today card shows "Od ostatniej lekcji: 4 dni ćwiczeń, 65 min"
