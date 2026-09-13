// Defines auth collection names, session cookie names, and browser auth constants.
package authconfig

import "time"

const (
	LearnersCollectionName    = "learners"
	TeachersCollectionName    = "teachers"
	LegacyUsersCollectionName = "users"
	LearnerNameField          = "name"
	TeacherNameField          = "name"
	AuthIntentHeader          = "X-Requested-With"
	AuthIntentValue           = "fetch"
	SessionDuration           = 12 * time.Hour
)
