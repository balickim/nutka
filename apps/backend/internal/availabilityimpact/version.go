// This file derives opaque preview versions from normalized scheduling state.
package availabilityimpact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

func stateVersion(input Input, policy scheduling.IntervalPolicy, policyVersion string, proposal Proposal, availability []scheduling.Interval, conflicts []NearTermConflict, effects []DistantEffect) string {
	canonical := struct {
		Policy        scheduling.IntervalPolicy
		PolicyVersion string
		Proposal      Proposal
		Availability  []scheduling.Interval
		Lessons       []versionLesson
		Occurrences   []versionOccurrence
		Conflicts     []NearTermConflict
		Effects       []DistantEffect
		Source        string
	}{Policy: policy, PolicyVersion: policyVersion, Proposal: proposal, Availability: availability, Lessons: versionLessons(input.NearTermLessons), Occurrences: versionOccurrences(input.DistantOccurrences), Conflicts: conflicts, Effects: effects, Source: input.StateVersion}
	encoded, _ := json.Marshal(canonical)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

type versionLesson struct {
	ID, Plan, State, Start, End string
}

type versionOccurrence struct {
	ID, State, Reason, Start, End string
}

func versionLessons(values []Lesson) []versionLesson {
	result := make([]versionLesson, 0, len(values))
	for _, value := range sortedLessons(values) {
		interval, _ := normalizeInterval(value.Interval)
		result = append(result, versionLesson{value.ID, string(value.Plan), string(value.State), interval.Start.Format(time.RFC3339Nano), interval.End.Format(time.RFC3339Nano)})
	}
	return result
}

func versionOccurrences(values []ContractOccurrence) []versionOccurrence {
	result := make([]versionOccurrence, 0, len(values))
	for _, value := range sortedOccurrences(values) {
		interval, _ := normalizeInterval(value.Interval)
		result = append(result, versionOccurrence{value.ID, string(value.State), value.OmissionReason, interval.Start.Format(time.RFC3339Nano), interval.End.Format(time.RFC3339Nano)})
	}
	return result
}

func sortedLessons(values []Lesson) []Lesson {
	result := append([]Lesson(nil), values...)
	sort.SliceStable(result, func(left, right int) bool { return result[left].ID < result[right].ID })
	return result
}

func sortedOccurrences(values []ContractOccurrence) []ContractOccurrence {
	result := append([]ContractOccurrence(nil), values...)
	sort.SliceStable(result, func(left, right int) bool { return result[left].ID < result[right].ID })
	return result
}
