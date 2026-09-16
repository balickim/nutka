// This file detects protected-interval conflicts for active lessons and their shared participants.
package scheduling

// DetectConflicts finds active lessons whose protected intervals intersect the proposed lesson.
func DetectConflicts(proposed Interval, teacherID, learnerID string, lessons []Lesson, options ...IntervalPolicy) []ParticipantConflict {
	if !proposed.Valid() {
		return nil
	}
	policy, err := resolveIntervalPolicy(options...)
	if err != nil {
		return nil
	}
	protected := proposed.Protected(policy.Buffer)
	conflicts := make([]ParticipantConflict, 0)
	for _, lesson := range lessons {
		if lesson.Status == Cancelled || !lesson.Interval.Valid() {
			continue
		}
		participantID, participantFound := matchingParticipant(teacherID, learnerID, lesson)
		if !participantFound || !protected.Intersects(lesson.Interval.Protected(policy.Buffer)) {
			continue
		}
		conflicts = append(conflicts, ParticipantConflict{ParticipantID: participantID, Lesson: lesson, Interval: lesson.Interval.Protected(policy.Buffer)})
	}
	return conflicts
}

func matchingParticipant(teacherID, learnerID string, lesson Lesson) (string, bool) {
	if teacherID != "" && lesson.TeacherID == teacherID {
		return teacherID, true
	}
	if learnerID != "" && lesson.LearnerID == learnerID {
		return learnerID, true
	}
	return "", false
}

func HasParticipantConflict(proposed Interval, teacherID, learnerID string, lessons []Lesson, options ...IntervalPolicy) bool {
	return len(DetectConflicts(proposed, teacherID, learnerID, lessons, options...)) > 0
}

func ValidateNoParticipantConflict(proposed Interval, teacherID, learnerID string, lessons []Lesson, options ...IntervalPolicy) error {
	if len(DetectConflicts(proposed, teacherID, learnerID, lessons, options...)) > 0 {
		return ErrConflict
	}
	return nil
}
