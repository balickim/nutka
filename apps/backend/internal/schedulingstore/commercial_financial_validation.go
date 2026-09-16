// Package schedulingstore validates ledger references and immutable policy snapshots.
// It keeps financial records assignment-scoped and rejects snapshots that omit authoritative policy values.
package schedulingstore

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func validateFinancialEntry(app core.App, entry *core.Record) error {
	assignment := entry.GetString(AssignmentField)
	for _, resolver := range []func(core.App, *core.Record) (string, error){chargeAssignment, packageAssignment, tokenAssignment, lessonAssignment, sourceAssignment} {
		value, err := resolver(app, entry)
		if err != nil {
			return err
		}
		if value != "" {
			if assignment != "" && assignment != value {
				return fmt.Errorf("financial entry %s references records from different assignments", entry.Id)
			}
			assignment = value
		}
	}
	return validateRelatedEntry(app, entry, assignment)
}

func relatedAssignment(app core.App, entry *core.Record, field, collection string) (string, error) {
	id := entry.GetString(field)
	if id == "" {
		return "", nil
	}
	record, err := findRelated(app, collection, id)
	if err != nil {
		return "", err
	}
	return record.GetString(AssignmentField), nil
}

func chargeAssignment(app core.App, entry *core.Record) (string, error) {
	return relatedAssignment(app, entry, ChargeField, ChargesCollectionName)
}

func packageAssignment(app core.App, entry *core.Record) (string, error) {
	return relatedAssignment(app, entry, RelatedPackageField, LessonPackagesCollectionName)
}

func lessonAssignment(app core.App, entry *core.Record) (string, error) {
	return relatedAssignment(app, entry, RelatedLessonField, LessonsCollectionName)
}

func tokenAssignment(app core.App, entry *core.Record) (string, error) {
	id := entry.GetString(RelatedTokenField)
	if id == "" {
		return "", nil
	}
	token, err := findRelated(app, PackageTokensCollectionName, id)
	if err != nil {
		return "", err
	}
	packageRecord, err := findRelated(app, LessonPackagesCollectionName, token.GetString(PackageField))
	if err != nil {
		return "", err
	}
	return packageRecord.GetString(AssignmentField), nil
}

func sourceAssignment(app core.App, entry *core.Record) (string, error) {
	sourceType, sourceID := entry.GetString(SourceTypeField), entry.GetString(SourceIDField)
	if sourceType == "" && sourceID == "" {
		return "", nil
	}
	if sourceType == "" || sourceID == "" {
		return "", fmt.Errorf("financial entry %s source type and source id are required together", entry.Id)
	}
	collection := map[string]string{"ad_hoc": LessonsCollectionName, "package": LessonPackagesCollectionName, "regular_contract": RegularContractsCollectionName}[sourceType]
	if collection == "" {
		return "", fmt.Errorf("financial entry %s has invalid source type %q", entry.Id, sourceType)
	}
	return relatedAssignment(app, entry, SourceIDField, collection)
}

func validateRelatedEntry(app core.App, entry *core.Record, assignment string) error {
	relatedID := entry.GetString(RelatedEntryField)
	if relatedID == "" {
		return nil
	}
	related, err := findRelated(app, FinancialEntriesCollectionName, relatedID)
	if err != nil {
		return err
	}
	if relatedAssignment := related.GetString(AssignmentField); relatedAssignment != "" && assignment != "" && relatedAssignment != assignment {
		return fmt.Errorf("financial entry %s and related entry %s must share assignment", entry.Id, related.Id)
	}
	return nil
}

func validatePolicySnapshot(record *core.Record) error {
	raw := record.GetRaw(PolicySnapshotField)
	snapshot, present, err := decodePolicySnapshot(raw)
	if err != nil {
		return fmt.Errorf("%s %s has invalid policy snapshot: %w", record.Collection().Name, record.Id, err)
	}
	if !present {
		return nil
	}
	if err := snapshot.Validate(); err != nil {
		return fmt.Errorf("%s %s has invalid policy snapshot: %w", record.Collection().Name, record.Id, err)
	}
	if version := record.GetString(PolicyVersionField); version != "" && version != snapshot.Version {
		return fmt.Errorf("%s %s policy version does not match its snapshot", record.Collection().Name, record.Id)
	}
	return nil
}

func decodePolicySnapshot(raw any) (businesspolicy.PolicySnapshot, bool, error) {
	if raw == nil {
		return businesspolicy.PolicySnapshot{}, false, nil
	}
	if value, ok := raw.(types.JSONRaw); ok {
		raw = []byte(value)
	}
	if value, ok := raw.(string); ok {
		if strings.TrimSpace(value) == "" {
			return businesspolicy.PolicySnapshot{}, false, nil
		}
		raw = []byte(value)
	}
	encoded, ok := raw.([]byte)
	if !ok {
		var err error
		encoded, err = json.Marshal(raw)
		if err != nil {
			return businesspolicy.PolicySnapshot{}, false, err
		}
	}
	if len(encoded) == 0 {
		return businesspolicy.PolicySnapshot{}, false, nil
	}
	var snapshot businesspolicy.PolicySnapshot
	err := json.Unmarshal(encoded, &snapshot)
	return snapshot, true, err
}
