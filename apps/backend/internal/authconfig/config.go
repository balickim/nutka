package authconfig

import "time"

const (
	LearnersCollectionName = "learners"
	LearnerNameField       = "name"
	SessionCookieName      = "__Host-nutka_session"
	AuthIntentHeader       = "X-Requested-With"
	AuthIntentValue        = "fetch"
	SessionDuration        = 12 * time.Hour
)
