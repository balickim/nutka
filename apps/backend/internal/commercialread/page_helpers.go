// This file keeps pagination and redaction helpers shared by focused read queries.
package commercialread

import (
	"math"
	"strings"

	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
)

func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = cloneValue(value)
	}
	return result
}

func cloneValue(value any) any {
	switch nested := value.(type) {
	case map[string]any:
		return cloneMap(nested)
	case []any:
		result := make([]any, len(nested))
		for index, item := range nested {
			result[index] = cloneValue(item)
		}
		return result
	default:
		return value
	}
}

func redact(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		if protectedKey(key) {
			continue
		}
		switch nested := value.(type) {
		case map[string]any:
			result[key] = redact(nested)
		case []any:
			items := make([]any, len(nested))
			for index, item := range nested {
				if child, ok := item.(map[string]any); ok {
					items[index] = redact(child)
				} else {
					items[index] = item
				}
			}
			result[key] = items
		default:
			result[key] = value
		}
	}
	return result
}

func protectedKey(key string) bool {
	switch strings.ToLower(key) {
	case "internal_note", "teacher_note", "teacher", "learner", "actor_id":
		return true
	default:
		return false
	}
}

func cloneStrings(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, text := range value {
		result[key] = text
	}
	return result
}

func safeRelatedIDs(value map[string]string) map[string]string {
	result := make(map[string]string)
	for key, text := range value {
		switch strings.ToLower(key) {
		case "assignment", "lesson", "package", "token", "contract", "charge", "financial_entry", "corrects_event":
			result[key] = text
		}
	}
	return cloneStrings(result)
}

func occurrencesViews(values []regularcontract.Occurrence) []ContractOccurrenceView {
	result := make([]ContractOccurrenceView, 0, len(values))
	for _, value := range values {
		result = append(result, occurrenceView(value))
	}
	return result
}

func validPage(page, perPage int) error {
	if page < 1 || perPage < 1 || perPage > 100 {
		return ErrPagination
	}
	return nil
}

func boundedItems(values []UnresolvedItem, page, perPage int) []UnresolvedItem {
	start, end := pageBounds(len(values), page, perPage)
	return values[start:end]
}

func pageOf[T any](values []T, page, perPage int) Page[T] {
	start, end := pageBounds(len(values), page, perPage)
	return Page[T]{Items: values[start:end], Page: page, PerPage: perPage, TotalItems: len(values), TotalPages: pages(len(values), perPage)}
}

func pageBounds(length, page, perPage int) (int, int) {
	start := (page - 1) * perPage
	if start > length {
		start = length
	}
	end := start + perPage
	if end > length {
		end = length
	}
	return start, end
}

func pages(length, perPage int) int {
	if length == 0 {
		return 0
	}
	return int(math.Ceil(float64(length) / float64(perPage)))
}
